package db

import log "unknwon.dev/clog/v2"

func prepareCICtx(ptask *PipeTask) (*CIContext, error) {
	repo := ptask.Pipeline.repoDB
	ctx := &CIContext{
		commit: ptask.Pipeline.Commit,
		config: ptask.Pipeline.Config,
		repo:   repo,
	}
	if repo.Owner == nil {
		if err := repo.GetOwner(); err != nil {
			return nil, err
		}
	}
	ctx.owner = repo.Owner
	return ctx, nil
}

func (ptask *PipeTask) CI001() error {

	context, err := prepareCICtx(ptask)
	if err != nil {
		return err
	}

	// load repo
	if err := ptask.LoadRepo(context); err != nil {
		return err
	}
	log.Info("load repo success for CI task: %d", ptask.ID)
	context.path = ptask.Pipeline.CIPath

	// static code validation
	if err := ptask.Validation(context); err != nil {
		return err
	}
	log.Info("static code validation success for CI task: %d", ptask.ID)

	// build image
	if err := ptask.Build(context); err != nil {
		return err
	}
	log.Info("build image success for CI task: %d", ptask.ID)

	// test
	// TODO

	// push image
	if err := ptask.Push(context); err != nil {
		return err
	}
	log.Info("push image success for CI task: %d", ptask.ID)

	// deploy image
	if err := ptask.Deploy(context); err != nil {
		return err
	}
	log.Info("deploy success for CI task: %d", ptask.ID)

	return nil
}
