package db

import log "unknwon.dev/clog/v2"

func (ptask *PipeTask) CI001() error {

	context := &CIContext{
		owner:  ptask.Pipeline.repoDB.MustOwner().LowerName,
		repo:   ptask.Pipeline.repoDB.LowerName,
		commit: ptask.Pipeline.Commit,
		config: ptask.Pipeline.Config,
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
	log.Info("build image for success for CI task: %d", ptask.ID)

	// test
	// TODO

	// push image
	if err := ptask.Push(context); err != nil {
		return err
	}
	log.Info("push image for success for CI task: %d", ptask.ID)

	return nil
}
