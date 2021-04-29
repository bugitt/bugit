package project

import (
	"net/http"

	log "unknwon.dev/clog/v2"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/context"
	"git.scs.buaa.edu.cn/iobs/bugit/internal/db"
)

// CreateOption 创建project时可以提供的参数
type CreateOption struct {
	Name       string `json:"projectName" form:"projectName" binding:"Required"`
	OrgName    string `json:"organizationName" form:"organizationName"`
	ExpName    string `json:"expName" form:"expName" binding:"Required"`
	ExpID      int64  `json:"expId" form:"expId" binding:"Required"`
	CourseName string `json:"courseName" form:"courseName" binding:"Required"`
	CourseID   int64  `json:"courseId" form:"courseId" binding:"Required"`
}

func GetAllProjects(c *context.APIContext) {
	projects, err := db.GetAllProjectsWithCoAndAttr(c.User)
	if err != nil {
		log.Error(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}
	c.JSONSuccess(projects)
}

func GetProjectsByCourse(c *context.APIContext) {
	courseIDList, err := db.GetCourseIDListByToken(c.Token())
	if err != nil {
		log.Error(err.Error())
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	projects, err := db.GetProjectsByCourseIDList(courseIDList)
	if err != nil {
		log.Error(err.Error())
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSONSuccess(projects)
}

func CreateProject(c *context.APIContext, form CreateOption) {
	senderID := c.User.ID
	if form.OrgName != "" {
		user, err := db.GetUserByName(form.OrgName)
		if err != nil && !db.IsErrUserNotExist(err) {
			log.Error(err.Error())
			c.JSON(http.StatusInternalServerError, "error: get org")
			return
		}
		if user == nil {
			// 创建新的org
			if !c.User.CanCreateOrganization() {
				c.JSON(http.StatusBadRequest, "error: this user can not create org")
				return
			}
			org := &db.User{
				Name:     form.OrgName,
				IsActive: true,
				Type:     db.UserOrganization,
			}
			if err := db.CreateOrganization(org, c.User); err != nil {
				c.JSON(http.StatusBadRequest, "error: create org")
				return
			}
			log.Trace("Organization created: %s", org.Name)

			senderID = org.ID
		} else {
			senderID = user.ID
		}
	}
	project := &db.Project{
		Name:       form.Name,
		SenderID:   senderID,
		ExpID:      form.ExpID,
		ExpString:  form.ExpName,
		CourseID:   form.CourseID,
		CourseName: form.CourseName,
	}
	if err := db.CreateProject(project); err != nil {
		log.Error(err.Error())
		c.JSON(http.StatusBadRequest, "please check for duplicate: (senderID, expID)")
		return
	}

	c.JSON(http.StatusCreated, project)
}

func ListMembers(c *context.APIContext) {
	project := c.Project.Project
	// 获取该project中的成员
	members, err := project.GetMembers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSONSuccess(members)
}

func ListRepos(c *context.APIContext) {
	project := c.Project.Project
	// 获取该project中的所有仓库
	repos, err := project.GetRepos()
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	for i := range repos {
		err = repos[i].LoadBranches()
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}
	}
	c.JSONSuccess(repos)
}

func GetDeploy(c *context.APIContext) {
	project := c.Project.Project

	// 获取该project中的所有仓库
	repos, err := project.GetRepos()
	deploys := make([]*db.DeployDes, 0, len(repos))
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	for _, repo := range repos {
		ds, err := db.GetDeploy(repo)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}
		deploys = append(deploys, ds)
	}

	c.JSONSuccess(deploys)
}

func CreateDeploy(c *context.APIContext, opt db.DeployOption) {
	var err error
	project := c.Project.Project
	opt.Repo, err = db.GetRepositoryByID(opt.RepoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	if opt.Repo.ProjectID != project.ID {
		c.JSON(http.StatusForbidden, "this repo doesn't belong to the project")
		return
	}

	// 好了，终于可以触发部署了
	opt.Pusher = c.User
	err = db.CreateDeploy(&opt)
	if err != nil {
		if db.IsErrNoNeedDeploy(err) || db.IsErrNoValidCIConfig(err) {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}

	c.Status(http.StatusOK)
}
