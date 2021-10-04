// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package org

import (
	log "unknwon.dev/clog/v2"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/auth"
	"git.scs.buaa.edu.cn/iobs/bugit/internal/conf"
	"git.scs.buaa.edu.cn/iobs/bugit/internal/context"
	"git.scs.buaa.edu.cn/iobs/bugit/internal/db"
	"git.scs.buaa.edu.cn/iobs/bugit/internal/form"
	"git.scs.buaa.edu.cn/iobs/bugit/internal/route/user"
)

const (
	SETTINGS_OPTIONS = "org/settings/options"
	SETTINGS_DELETE  = "org/settings/delete"
)

func Settings(c *context.Context) {
	c.Title("org.settings")
	c.Data["PageIsSettingsOptions"] = true
	c.Success(SETTINGS_OPTIONS)
}

func SettingsPost(c *context.Context, f form.UpdateOrgSetting) {
	c.Title("org.settings")
	c.Data["PageIsSettingsOptions"] = true

	if c.HasError() {
		c.Success(SETTINGS_OPTIONS)
		return
	}

	org := c.Org.Organization

	if c.User.IsAdmin {
		org.MaxRepoCreation = f.MaxRepoCreation
	}

	org.FullName = f.FullName
	org.Description = f.Description
	org.Website = f.Website
	org.Location = f.Location
	if err := db.UpdateUser(org); err != nil {
		c.Error(err, "update user")
		return
	}
	log.Trace("Organization setting updated: %s", org.Name)
	c.Flash.Success(c.Tr("org.settings.update_setting_success"))
	c.Redirect(c.Org.OrgLink + "/settings")
}

func SettingsAvatar(c *context.Context, f form.Avatar) {
	f.Source = form.AVATAR_LOCAL
	if err := user.UpdateAvatarSetting(c, f, c.Org.Organization); err != nil {
		c.Flash.Error(err.Error())
	} else {
		c.Flash.Success(c.Tr("org.settings.update_avatar_success"))
	}

	c.Redirect(c.Org.OrgLink + "/settings")
}

func SettingsDeleteAvatar(c *context.Context) {
	if err := c.Org.Organization.DeleteAvatar(); err != nil {
		c.Flash.Error(err.Error())
	}

	c.Redirect(c.Org.OrgLink + "/settings")
}

func SettingsDelete(c *context.Context) {
	c.Title("org.settings")
	c.PageIs("SettingsDelete")

	org := c.Org.Organization
	if c.Req.Method == "POST" {
		if _, err := db.Users.Authenticate(c.User.Name, c.Query("password"), c.User.LoginSource); err != nil {
			if auth.IsErrBadCredentials(err) {
				c.RenderWithErr(c.Tr("form.enterred_invalid_password"), SETTINGS_DELETE, nil)
			} else {
				c.Error(err, "authenticate user")
			}
			return
		}

		if err := db.DeleteOrganization(org); err != nil {
			if db.IsErrUserOwnRepos(err) {
				c.Flash.Error(c.Tr("form.org_still_own_repo"))
				c.Redirect(c.Org.OrgLink + "/settings/delete")
			} else {
				c.Error(err, "delete organization")
			}
		} else {
			log.Trace("Organization deleted: %s", org.Name)
			c.Redirect(conf.Server.Subpath + "/")
		}
		return
	}

	c.Success(SETTINGS_DELETE)
}
