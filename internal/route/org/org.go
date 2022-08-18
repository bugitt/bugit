// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package org

import (
	log "unknwon.dev/clog/v2"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/context"
	"git.scs.buaa.edu.cn/iobs/bugit/internal/db"
	"git.scs.buaa.edu.cn/iobs/bugit/internal/form"
)

const (
	CREATE = "org/create"
	EXPS   = "org/exp_items"
)

func Create(c *context.Context) {
	c.Title("new_org")
	c.Success(CREATE)
}

func CreatePost(c *context.Context, f form.CreateOrg) {
	c.Title("new_org")

	if c.HasError() {
		c.Success(CREATE)
		return
	}

	org := &db.User{
		Name:     f.OrgName,
		IsActive: true,
		Type:     db.UserOrganization,
	}

	if err := db.CreateOrganization(org, c.User); err != nil {
		switch {
		case db.IsErrUserAlreadyExist(err):
			c.Data["Err_OrgName"] = true
			c.RenderWithErr(c.Tr("form.org_name_been_taken"), CREATE, &f)
		case db.IsErrNameNotAllowed(err):
			c.Data["Err_OrgName"] = true
			c.RenderWithErr(c.Tr("org.form.name_not_allowed", err.(db.ErrNameNotAllowed).Value()), CREATE, &f)
		case db.IsErrUserExpConflict(err):
			c.Data["Err_OrgExps"] = true
			c.Data["Err_OrgCourses"] = true
			c.RenderWithErr(err.Error(), CREATE, &f)
		case db.IsErrUserOrgExpConflict(err):
			c.Data["Err_OrgExps"] = true
			c.Data["Err_OrgCourses"] = true
			c.RenderWithErr(c.Tr("form.project_for_exp_already_have"), CREATE, &f)
		default:
			c.Data["Err_OrgName"] = true
			c.Error(err, "create organization")
		}
		return
	}
	log.Trace("Organization created: %s", org.Name)

	c.RedirectSubpath("/org/" + f.OrgName + "/dashboard")
}
