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
		repos, err := c.Project.Project.GetRepos()
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}
		c.Data["Repos"] = repos
	}
	c.Success(HomeView)
}
