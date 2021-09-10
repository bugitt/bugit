package ci

import (
	"context"
	"path/filepath"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/conf"
	"git.scs.buaa.edu.cn/iobs/bugit/internal/db"
	"gopkg.in/yaml.v3"
)

type Context struct {
	context.Context
	pusher            *db.User
	owner             *db.User
	repo              *db.Repository
	path              string
	imageTag          []string
	harborProjectName string
	commit            string
	refName           string
	config            *db.CIConfig
	pipeline          *db.Pipeline
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

	config := &db.CIConfig{}
	err = yaml.Unmarshal([]byte(p.ConfigString), config)
	if err != nil {
		return nil, err
	}

	return &Context{
		Context:           c,
		pusher:            pusher,
		owner:             repo.Owner,
		repo:              repo,
		path:              filepath.Join(conf.Devops.Tmpdir, repo.MustOwner().Name, repo.Name, p.Commit),
		imageTag:          []string{p.ImageTag},
		harborProjectName: p.HarborProjectName,
		commit:            p.Commit,
		refName:           p.RefName,
		config:            config,
		pipeline:          p,
	}, nil
}

func (ctx *Context) updateStage(stage db.PipeStage, taskNum int) error {
	return ctx.pipeline.UpdateStage(stage, taskNum)
}
