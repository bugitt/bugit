package ci

import log "unknwon.dev/clog/v2"

func run(ctx *Context) (err error) {
	// load repo
	err = loadRepo(ctx)
	if err != nil {
		log.Error("load repo failed: %d, error message: %s", ctx.pipeline.ID, err.Error())
		return
	}
	log.Info("load repo successfully: %d", ctx.pipeline.ID)

	// pre build
	err = preBuild(ctx)
	if err != nil {
		log.Error("pre build failed: %d, error message: %s", ctx.pipeline.ID, err.Error())
		return
	}
	log.Info("pre build successfully: %d", ctx.pipeline.ID)

	if ctx.config.Build == nil {
		return nil
	}

	// build
	err = build(ctx)
	if err != nil {
		log.Error("build failed: %d, error message: %s", ctx.pipeline.ID, err.Error())
		return
	}
	log.Info("build successfully: %d", ctx.pipeline.ID)

	// post build
	err = postBuild(ctx)
	if err != nil {
		log.Error("post build failed: %d, error message: %s", ctx.pipeline.ID, err.Error())
		return
	}
	log.Info("post build successfully: %d", ctx.pipeline.ID)

	// push image
	err = push(ctx)
	if err != nil {
		log.Error("push image failed: %d, error message: %s", ctx.pipeline.ID, err.Error())
		return
	}
	log.Info("push image successfully: %d", ctx.pipeline.ID)

	// deploy
	err = deploy(ctx)
	if err != nil {
		log.Error("deploy failed: %d, error message: %s", ctx.pipeline.ID, err.Error())
		return
	}
	log.Info("deploy successfully: %d", ctx.pipeline.ID)

	return nil
}
