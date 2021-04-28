package db

import (
	"context"
	"fmt"

	log "unknwon.dev/clog/v2"
)

func prepareCICtx(ptask *PipeTask, c context.Context) (*CIContext, error) {
	repo := ptask.Pipeline.repoDB
	ctx := &CIContext{
		commit:   ptask.Pipeline.Commit,
		config:   ptask.Pipeline.Config,
		repo:     repo,
		refName:  ptask.Pipeline.RefName,
		Context:  c,
		imageTag: ptask.ImageTag,
	}
	if repo.Owner == nil {
		if err := repo.GetOwner(); err != nil {
			return nil, err
		}
	}
	ctx.owner = repo.Owner
	return ctx, nil
}

func (ptask *PipeTask) CI001(c context.Context) error {
	context, err := prepareCICtx(ptask, c)
	if err != nil {
		return fmt.Errorf("prepare ci context error: %w", err)
	}

	// load repo
	if err := ptask.LoadRepo(context); err != nil {
		return fmt.Errorf("load repo error: %w", err)
	}
	log.Info("load repo success for CI task: %d", ptask.ID)
	context.path = ptask.Pipeline.CIPath

	// static code validation
	if err := ptask.Validation(context); err != nil {
		return fmt.Errorf("validate error: %w", err)
	}
	log.Info("static code validation success for CI task: %d", ptask.ID)

	// build image
	if err := ptask.Build(context); err != nil {
		return fmt.Errorf("build error: %w", err)
	}
	log.Info("build image success for CI task: %d", ptask.ID)

	// test
	// TODO

	// push image
	if err := ptask.Push(context); err != nil {
		return fmt.Errorf("push image error: %w", err)
	}
	log.Info("push image success for CI task: %d", ptask.ID)

	// deploy image
	if err := ptask.Deploy(context); err != nil {
		return fmt.Errorf("deploy to kubernetes error: %w", err)
	}
	log.Info("deploy success for CI task: %d", ptask.ID)

	return nil
}
