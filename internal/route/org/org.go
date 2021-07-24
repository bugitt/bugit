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
	courseID, request_exps := c.QueryInt64("course_id"), c.QueryBool("request_exps")
	if courseID > 0 && request_exps {
		exps, err := db.GetExpsByCourseID(courseID)
		if err != nil {
			c.Error(err, "get exps error")
			return
		}
		c.Data["Exps"] = exps
		c.Success(EXPS)
		return
	}

	c.Title("new_org")

	// get all courses for this user
	courses, err := db.GetCoursesByStudentID(c.User.StudentID)
	if err != nil {
		c.Error(err, "get courses error")
		return
	}
	c.Data["Courses"] = courses

	c.Success(CREATE)
}

func CreatePost(c *context.Context, f form.CreateOrg) {
	c.Title("new_org")

	if c.HasError() {
		c.Success(CREATE)
		return
	}

	course, err := db.GetCourseByID(f.OrgCourse)
	if err != nil {
		c.Error(err, "get course")
		return
	}
	exp, err := db.GetExpByID(f.OrgExp)
	if err != nil {
		c.Error(err, "get experiment")
		return
	}

	org := &db.User{
		Name:     f.OrgName,
		IsActive: true,
		Type:     db.UserOrganization,

		CourseID:   course.ID,
		CourseName: course.Name,
		ExpID:      exp.ID,
		ExpName:    exp.Name,
	}

	if err := db.CreateOrganization(org, c.User); err != nil {
		// get all courses for this user 用于再次渲染前端页面
		if courses, err := db.GetCoursesByStudentID(c.User.StudentID); err != nil {
			c.Error(err, "get courses error")
			return
		} else {
			c.Data["Courses"] = courses
		}

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
