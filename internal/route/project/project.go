package project

import (
	"fmt"
	"math/rand"
	"net/http"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/context"
	"git.scs.buaa.edu.cn/iobs/bugit/internal/db"
	"git.scs.buaa.edu.cn/iobs/bugit/internal/form"

	log "unknwon.dev/clog/v2"
)

const (
	HomeView = "project/home"
	CREATE   = "project/create"
)

func Home(c *context.Context) {
	tab := c.Query("tab")
	c.Data["TabName"] = tab
	if tab == "" || tab == "repo" {
		err := c.Project.Repos.LoadAttributes()
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}
		c.Data["Repos"] = c.Project.Repos
	} else if tab == "pipeline" {
		// TODO: 后续考虑不用一次加载完
		depList, err := c.Project.Project.GetDeployList(c.Project.Repos...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}
		c.Data["PipeList"] = depList
	}
	c.Success(HomeView)
}

func Create(c *context.Context) {
	c.Title("new_org")
	c.Success(CREATE)
}

func CreatePost(c *context.Context, f form.CreateProject) {
	c.Title("new_project")
	if c.HasError() {
		c.Success(CREATE)
		return
	}
	senderID := c.User.ID
	project := &db.Project{
		Name:     f.ProjectName,
		SenderID: senderID,

		// TODO: 修复假参数
		ExpID:      rand.Int63n(100000000000),
		ExpString:  "随机测试实验",
		CourseID:   rand.Int63n(10000000000),
		CourseName: "随机测试课程",
	}
	if err := db.CreateProject(project); err != nil {
		log.Error(err.Error())
		c.Data["Err_Project"] = true
		c.RenderWithErr(c.Tr("error create project"), CREATE, &f)
		return
	}
	log.Trace("Project created: %s", project.Name)

	c.RedirectSubpath(fmt.Sprintf("/project/%d", project.ID))
}
