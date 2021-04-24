package project

import (
	"net/http"

	jsoniter "github.com/json-iterator/go"
	log "unknwon.dev/clog/v2"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/context"
	"git.scs.buaa.edu.cn/iobs/bugit/internal/db"
)

// CreateOption 创建project时可以提供的参数
type CreateOption struct {
	Name       string `json:"projectName" binding:"Required"`
	OrgName    string `json:"organizationName"`
	ExpName    string `json:"expName" binding:"Required"`
	ExpID      int64  `json:"expId" binding:"Required"`
	CourseName string `json:"courseName" binding:"Required"`
	CourseID   int64  `json:"courseId" binding:"Required"`
	IsNewOrg   bool   `json:"isNewOrganization"`
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
	courseListS := c.Query("courseIds")
	var (
		projects []*db.Project
		err      error
	)
	if courseListS == "-1" {
		projects, err = db.GetUserAllProjects(c.User)
		if err != nil {
			log.Error(err.Error())
			c.Status(http.StatusInternalServerError)
			return
		}
		c.JSONSuccess(projects)
	}
	courseIDList := make([]int64, 0)
	err = jsoniter.Unmarshal([]byte(courseListS), &courseIDList)
	if err != nil {
		log.Error(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}
	projects, err = db.GetProjectsByUserAndCourse(c.UserID(), courseIDList)
	if err != nil {
		log.Error(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}
	c.JSONSuccess(projects)
}

func CreateProject(c *context.APIContext, form CreateOption) {
	if form.IsNewOrg {
		// 创建新的org
		if form.OrgName == "" {
			c.Status(http.StatusBadRequest)
			return
		}
		if !c.User.CanCreateOrganization() {
			c.Status(http.StatusBadRequest)
			return
		}
		org := &db.User{
			Name:     form.OrgName,
			IsActive: true,
			Type:     db.UserOrganization,
		}
		if err := db.CreateOrganization(org, c.User); err != nil {
			c.Status(http.StatusBadRequest)
			return
		}
		log.Trace("Organization created: %s", org.Name)
	}

	senderID := c.User.ID
	if form.OrgName != "" {
		user, err := db.GetUserByName(form.OrgName)
		if err != nil {
			log.Error(err.Error())
			c.Status(http.StatusInternalServerError)
			return
		}
		if user != nil {
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
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusCreated, project)
}
