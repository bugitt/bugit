package context

import (
	"git.scs.buaa.edu.cn/iobs/bugit/internal/db"
	"gopkg.in/macaron.v1"
)

type Project struct {
	Project *db.Project
}

func ProjectAssignment() macaron.Handler {
	return func(c *Context) {
		project, err := db.GetProjectByID(c.ParamsInt64(":projectID"))
		if err != nil || project == nil {
			if err != nil {
				c.NotFoundOrError(err, "get project by id")
				return
			}
		}
		c.Project = &Project{
			Project: project,
		}
	}
}

func AuthProjectUser() macaron.Handler {
	return func(c *Context) {

	}
}
