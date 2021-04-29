package project

import "git.scs.buaa.edu.cn/iobs/bugit/internal/context"

const (
	HomeView = "project/home"
)

func Home(c *context.Context) {
	c.Success(HomeView)
}
