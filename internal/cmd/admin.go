// Copyright 2016 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cmd

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/kube"
	"git.scs.buaa.edu.cn/iobs/bugit/internal/platform"
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
			subcmdSyncKS,
			subcmdSyncUsersWithCloudPlatform,
		},
	}

	subcmdCreateUser = cli.Command{
		Name:   "create-user",
		Usage:  "Create a new user in database",
		Action: runCreateUser,
		Flags: []cli.Flag{
			stringFlag("password", "", "User password"),
			stringFlag("email", "", "User email address"),
			stringFlag("studentID", "", "Student number"),
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

	subcmdSyncKS = cli.Command{
		Name:   "sync-ks",
		Usage:  "sync users and orgs to ks",
		Action: adminSyncKS,
		Flags: []cli.Flag{
			boolFlag("all", "sync all users and orgs"),
			stringFlag("username", "", "user name"),
		},
	}

	subcmdSyncUsersWithCloudPlatform = cli.Command{
		Name:   "sync-users-with-cloud-platform",
		Usage:  "sync users with cloud platform",
		Action: adminSyncCloud,
	}
)

func runCreateUser(c *cli.Context) error {
	if !c.IsSet("studentID") {
		return errors.New("StudentID is not specified")
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

	platform.Init()

	if _, err = db.SetEngine(); err != nil {
		return errors.Wrap(err, "set engine")
	}

	studentID := strings.ToLower(c.String("studentID"))
	if err := db.CreateUser(&db.User{
		Name:      studentID,
		Email:     c.String("email"),
		Passwd:    c.String("password"),
		StudentID: studentID,
		IsActive:  true,
		IsAdmin:   c.Bool("admin"),
	}); err != nil {
		return fmt.Errorf("CreateUser: %v", err)
	}

	fmt.Printf("New user '%s' has been successfully created!\n", studentID)
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

func adminSyncKS(c *cli.Context) error {
	err := conf.Init(c.String("config"))
	if err != nil {
		return errors.Wrap(err, "init configuration")
	}
	conf.InitLogging(true)

	if _, err = db.SetEngine(); err != nil {
		return errors.Wrap(err, "set engine")
	}

	platform.Init()

	var users []*db.User
	isAll := c.Bool("all")
	if isAll {
		uList, err := db.GetAllUsersAndOrgs()
		if err != nil {
			fmt.Println(err.Error())
			return err
		}
		users = uList
	} else {
		name := c.String("username")
		u, err := db.GetUserByName(name)
		if err != nil {
			fmt.Println(err.Error())
			return err
		}
		users = append(users, u)
	}

	// 处理user
	for _, u := range users {
		if u.IsOrganization() {
			continue
		}
		err := func() error {
			_, _, err := platform.CreateKSUser(u.Name, u.Email, "")
			return err
		}()
		if err == nil {
			fmt.Printf("crated ks user %s\n", u.Name)
		} else {
			fmt.Printf("crated ks user %s, failed\n", u.Name)
		}
	}

	// 处理org
	for _, org := range users {
		if !org.IsOrganization() {
			continue
		}
		err := func() error {
			_, err := platform.CreateKSProject("admin", org.LowerName)
			if err != nil {
				return err
			}
			err = org.GetMembers(-1)
			if err != nil {
				return err
			}
			for _, u := range org.Members {
				err = platform.AddKSOwner(u.Name, org.LowerName)
				if err != nil {
					fmt.Printf("add %s to %s, failed", u.Name, org.LowerName)
					err = nil
				}
			}
			return nil
		}()
		if err == nil {
			fmt.Printf("crated ks org %s\n", org.Name)
		} else {
			fmt.Printf("crated ks org %s, failed\n", org.Name)
		}
	}
	return nil
}

func adminSyncCloud(c *cli.Context) error {
	err := conf.Init(c.String("config"))
	if err != nil {
		return errors.Wrap(err, "init configuration")
	}
	conf.InitLogging(true)
	if _, err := db.SetEngine(); err != nil {
		return errors.Wrap(err, "set engine")
	}

	platform.Init()

	uList, err := db.GetAllUsersAndOrgs()
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	for _, u := range uList {
		if u.Type == db.UserIndividual {
			err := kube.AddProjectMember(context.Background(), "project-"+u.Name, u.Name, "operator")
			if err != nil {
				return err
			}
			fmt.Println(u.Name)
		}
	}

	//uMap := make(map[string]*db.User)
	//for _, u := range uList {
	//	uMap[u.Name] = u
	//}

	//
	//cloudUList, err := db.GetAllCloudUserList()
	//if err != nil {
	//	return err
	//}
	//cloudUMap := make(map[string]*db.CloudUser)
	//for _, u := range cloudUList {
	//	cloudUMap[strings.ToLower(u.ID)] = u
	//}
	//
	//// 首先同步BuGit中的邮箱
	////for _, u := range uList {
	////	if cloudU, ok := cloudUMap[u.Name]; ok && cloudU.Email != u.Email {
	////		cloudU.Email = u.Email
	////		if err = db.UpdateCloudUserEmail(cloudU); err != nil {
	////			return err
	////		}
	////	}
	////}
	//
	//validEmail := func(email string) bool {
	//	_, err := mail.ParseAddress(email)
	//	return err == nil
	//}
	//
	//{
	//	// clear duplicate emails
	//	m := make(map[string]bool)
	//	newList := make([]*db.CloudUser, 0)
	//	for _, u := range uList {
	//		if u.Type != db.UserOrganization {
	//			m[u.Email] = true
	//		}
	//	}
	//	for _, u := range cloudUList {
	//		if _, ok := m[u.Email]; !ok {
	//			newList = append(newList, u)
	//			m[u.Email] = true
	//		}
	//	}
	//	cloudUList = newList
	//}
	//
	//// 然后同步用户
	//for _, cloudU := range cloudUList {
	//	if _, ok := uMap[strings.ToLower(cloudU.ID)]; !ok && cloudU.Email != "" && cloudU.Email != "1@roycent.cn" && cloudU.Email != "unknown@buaa.edu.cn" && validEmail(cloudU.Email) {
	//		realCloudU := *cloudU
	//		fmt.Println(realCloudU.ID)
	//		studentID := strings.ToLower(realCloudU.ID)
	//		if err := db.CreateUser(&db.User{
	//			Name:      studentID,
	//			Email:     realCloudU.Email,
	//			Passwd:    conf.Harbor.DefaultPasswd,
	//			StudentID: studentID,
	//			IsActive:  true,
	//			IsAdmin:   false,
	//		}); err != nil {
	//			fmt.Printf("CreateUser: %#v\n, %#v", err, realCloudU)
	//			return err
	//		}
	//	}
	//}

	return nil
}
