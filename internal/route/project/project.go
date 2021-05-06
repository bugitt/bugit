package project

import (
	"net/http"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/context"
)

const (
	HomeView = "project/home"
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
