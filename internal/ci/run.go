package ci

func run(ctx *Context) (err error) {
	// load repo
	err = loadRepo(ctx)
	if err != nil {
		return
	}

	return nil
}
