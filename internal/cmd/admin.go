// Copyright 2016 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"
	"reflect"
	"runtime"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"gopkg.in/yaml.v3"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/conf"
	"git.scs.buaa.edu.cn/iobs/bugit/internal/db"
)

var (
	Admin = cli.Command{
		Name:  "admin",
		Usage: "Perform admin operations on command line",
		Description: `Allow using internal logic of Gogs without hacking into the source code
to make automatic initialization process more smoothly`,
		Subcommands: []cli.Command{
			subcmdCreateUser,
			subcmdDeleteInactivateUsers,
			subcmdDeleteRepositoryArchives,
			subcmdDeleteMissingRepositories,
			subcmdGitGcRepos,
			subcmdRewriteAuthorizedKeys,
			subcmdSyncRepositoryHooks,
			subcmdReinitMissingRepositories,
			subcmdMigrageFromSqlite,
			subcmdCIConfigExample,
		},
	}

	subcmdCreateUser = cli.Command{
		Name:   "create-user",
		Usage:  "Create a new user in database",
		Action: runCreateUser,
		Flags: []cli.Flag{
			stringFlag("name", "", "Username"),
			stringFlag("password", "", "User password"),
			stringFlag("email", "", "User email address"),
			boolFlag("admin", "User is an admin"),
			stringFlag("config, c", "", "Custom configuration file path"),
		},
	}

	subcmdDeleteInactivateUsers = cli.Command{
		Name:  "delete-inactive-users",
		Usage: "Delete all inactive accounts",
		Action: adminDashboardOperation(
			db.DeleteInactivateUsers,
			"All inactivate accounts have been deleted successfully",
		),
		Flags: []cli.Flag{
			stringFlag("config, c", "", "Custom configuration file path"),
		},
	}

	subcmdDeleteRepositoryArchives = cli.Command{
		Name:  "delete-repository-archives",
		Usage: "Delete all repositories archives",
		Action: adminDashboardOperation(
			db.DeleteRepositoryArchives,
			"All repositories archives have been deleted successfully",
		),
		Flags: []cli.Flag{
			stringFlag("config, c", "", "Custom configuration file path"),
		},
	}

	subcmdDeleteMissingRepositories = cli.Command{
		Name:  "delete-missing-repositories",
		Usage: "Delete all repository records that lost Git files",
		Action: adminDashboardOperation(
			db.DeleteMissingRepositories,
			"All repositories archives have been deleted successfully",
		),
		Flags: []cli.Flag{
			stringFlag("config, c", "", "Custom configuration file path"),
		},
	}

	subcmdGitGcRepos = cli.Command{
		Name:  "collect-garbage",
		Usage: "Do garbage collection on repositories",
		Action: adminDashboardOperation(
			db.GitGcRepos,
			"All repositories have done garbage collection successfully",
		),
		Flags: []cli.Flag{
			stringFlag("config, c", "", "Custom configuration file path"),
		},
	}

	subcmdRewriteAuthorizedKeys = cli.Command{
		Name:  "rewrite-authorized-keys",
		Usage: "Rewrite '.ssh/authorized_keys' file (caution: non-Gogs keys will be lost)",
		Action: adminDashboardOperation(
			db.RewriteAuthorizedKeys,
			"All public keys have been rewritten successfully",
		),
		Flags: []cli.Flag{
			stringFlag("config, c", "", "Custom configuration file path"),
		},
	}

	subcmdSyncRepositoryHooks = cli.Command{
		Name:  "resync-hooks",
		Usage: "Resync pre-receive, update and post-receive hooks",
		Action: adminDashboardOperation(
			db.SyncRepositoryHooks,
			"All repositories' pre-receive, update and post-receive hooks have been resynced successfully",
		),
		Flags: []cli.Flag{
			stringFlag("config, c", "", "Custom configuration file path"),
		},
	}

	subcmdReinitMissingRepositories = cli.Command{
		Name:  "reinit-missing-repositories",
		Usage: "Reinitialize all repository records that lost Git files",
		Action: adminDashboardOperation(
			db.ReinitMissingRepositories,
			"All repository records that lost Git files have been reinitialized successfully",
		),
		Flags: []cli.Flag{
			stringFlag("config, c", "", "Custom configuration file path"),
		},
	}

	subcmdMigrageFromSqlite = cli.Command{
		Name:   "migrate-sqlite",
		Usage:  "migrate data from sqlite3",
		Action: adminMigrateFromSqlite,
		Flags: []cli.Flag{
			stringFlag("sqlite, s", "", "sqlite db path"),
			stringFlag("config, c", "", "Custom configuration file path"),
		},
	}

	subcmdCIConfigExample = cli.Command{
		Name:   "ci-config-example",
		Usage:  "get ci config example",
		Action: getCIConfigExample,
	}
)

func runCreateUser(c *cli.Context) error {
	if !c.IsSet("name") {
		return errors.New("Username is not specified")
	} else if !c.IsSet("password") {
		return errors.New("Password is not specified")
	} else if !c.IsSet("email") {
		return errors.New("Email is not specified")
	}

	err := conf.Init(c.String("config"))
	if err != nil {
		return errors.Wrap(err, "init configuration")
	}
	conf.InitLogging(true)

	if _, err = db.SetEngine(); err != nil {
		return errors.Wrap(err, "set engine")
	}

	if err := db.CreateUser(&db.User{
		Name:     c.String("name"),
		Email:    c.String("email"),
		Passwd:   c.String("password"),
		IsActive: true,
		IsAdmin:  c.Bool("admin"),
	}); err != nil {
		return fmt.Errorf("CreateUser: %v", err)
	}

	fmt.Printf("New user '%s' has been successfully created!\n", c.String("name"))
	return nil
}

func adminDashboardOperation(operation func() error, successMessage string) func(*cli.Context) error {
	return func(c *cli.Context) error {
		err := conf.Init(c.String("config"))
		if err != nil {
			return errors.Wrap(err, "init configuration")
		}
		conf.InitLogging(true)

		if _, err = db.SetEngine(); err != nil {
			return errors.Wrap(err, "set engine")
		}

		if err := operation(); err != nil {
			functionName := runtime.FuncForPC(reflect.ValueOf(operation).Pointer()).Name()
			return fmt.Errorf("%s: %v", functionName, err)
		}

		fmt.Printf("%s\n", successMessage)
		return nil
	}
}

func adminMigrateFromSqlite(c *cli.Context) error {
	if !c.IsSet("sqlite") {
		return errors.New("sqlite is not specified")
	}

	err := conf.Init(c.String("config"))
	if err != nil {
		return errors.Wrap(err, "init configuration")
	}
	conf.InitLogging(true)

	if _, err = db.SetEngine(); err != nil {
		return errors.Wrap(err, "set engine")
	}

	if err := db.MigrateFromSqlite(c.String("sqlite")); err != nil {
		return fmt.Errorf("migrate from sqlite: %v", err)
	}

	fmt.Print("migrate from sqlite successfully")
	return nil
}

func getCIConfigExample(c *cli.Context) error {
	config := &db.CIConfig{}
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
