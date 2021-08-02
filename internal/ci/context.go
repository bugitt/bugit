package ci

import (
	"context"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/db"
)

type Context struct {
	context.Context
	path     string
	imageTag string
	owner    *db.User
	repo     *db.Repository
	commit   string
	refName  string
	config   *Config
	pipeline *db.Pipeline
}

func prepareCtx(pipeline *db.Pipeline) (*Context, error) {
	return &Context{}, nil
}
