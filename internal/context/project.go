package context

import (
	"net/http"
	"time"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/conf"
	"git.scs.buaa.edu.cn/iobs/bugit/internal/db"
	"gopkg.in/macaron.v1"
)

type IsProjectAdmin int

const (
	ProjectAdminNotSure IsProjectAdmin = iota
	ProjectAdminTrue
	ProjectAdminFalse
)

type Project struct {
	Project           *db.Project
	SenderProfileLink string
	IsProjectAdmin    IsProjectAdmin
}

func AuthProjectUser() macaron.Handler {
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

		authOK, err := authForAccessProject(c, project)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}
		if !authOK {
			c.JSON(http.StatusForbidden, "no permission to read the content of this project")
			return
		}

	}
}

func ProjectAssignment() macaron.Handler {
	return func(c *Context) {
		_ = c.Project.Project.LoadAttributes()
		c.Project.SenderProfileLink = conf.Server.Subpath + "/" + c.Project.Project.Sender.Name
		c.Data["Project"] = c.Project.Project
		c.Data["SenderProfileLink"] = c.Project.SenderProfileLink
		c.Data["Created"] = time.Unix(c.Project.Project.CreatedUnix, 0)
	}
}

func authForAccessProject(c *Context, project *db.Project) (bool, error) {
	// 如果该project是个人的项目
	if c.UserID() == project.SenderID {
		return true, nil
	}

	// 如果该project是组织的项目
	if org := project.Sender; org.IsOrganization() && org.IsOrgMember(c.UserID()) {
		return true, nil
	}

	// 如果是管理员
	return checkProjectCloudAdmin(c, project)
}

// checkProjectCloudAdmin 该操作非常耗时，请谨慎调用
func checkProjectCloudAdmin(c *Context, project *db.Project) (ok bool, err error) {
	if c.Project.IsProjectAdmin != ProjectAdminNotSure {
		return c.Project.IsProjectAdmin == ProjectAdminTrue, nil
	}
	courseIDList, err := db.GetCourseIDListByToken(c.Token())
	if err != nil {
		return false, err
	}
	for _, courseID := range courseIDList {
		if project.CourseID == courseID {
			ok = true
			break
		}
	}
	if ok {
		c.Project.IsProjectAdmin = ProjectAdminTrue
	} else {
		c.Project.IsProjectAdmin = ProjectAdminFalse
	}
	return
}
