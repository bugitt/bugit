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

	return nil
}
