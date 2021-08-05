package ci

import (
	"context"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/db"
)

type Context struct {
	context.Context
	pusher   *db.User
	owner    *db.User
	repo     *db.Repository
	path     string
	imageTag []string
	commit   string
	refName  string
	config   *db.CIConfig
	pipeline *db.Pipeline
}

func prepareCtx(c context.Context, p *db.Pipeline) (*Context, error) {
	repo, err := db.GetRepositoryByID(p.RepoID)
	if err != nil {
		return nil, err
	}
	if err = repo.LoadAttributes(); err != nil {
		return nil, err
	}

	pusher, err := db.GetUserByID(p.PusherID)
	if err != nil {
		return nil, err
	}

	return &Context{
		Context:  c,
		pusher:   pusher,
		owner:    repo.Owner,
		repo:     repo,
		path:     "",
		imageTag: []string{p.ImageTag},
		commit:   p.Commit,
		refName:  p.RefName,
		config:   nil,
		pipeline: p,
	}, nil
}
