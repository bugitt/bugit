package ci

import (
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/conf"
	"git.scs.buaa.edu.cn/iobs/bugit/internal/db"
	"github.com/loheagn/loclo/docker/image"
	"github.com/pkg/errors"
	log "unknwon.dev/clog/v2"
)

func build(ctx *Context) (err error) {
	err = ctx.updateStage(db.BuildStart, -1)
	if err != nil {
		return
	}

	config := ctx.config.Build
	switch strings.ToLower(config.Type) {
	case "docker":
		err = dockerBuild(ctx)
	default:
		err = errors.New("not support build type")
	}
	if err != nil {
		return
	}

	return ctx.updateStage(db.BuildEnd, -1)
}

func dockerBuild(ctx *Context) (err error) {
	var (
		outputLog string
		begin     = time.Now()
		result    = db.BuildResult{
			BasicTaskResult: db.BasicTaskResult{
				PipelineID: ctx.pipeline.ID,
			},
		}
	)

	defer func() {
		result.End(begin, err, outputLog)
		for _, tag := range ctx.imageTag {
			result.ImageTag += tag + ", "
		}
		dbErr := db.SaveCIResult(result)
		if dbErr != nil {
			if err != nil {
				log.Error("save build result failed, error message: %s", dbErr.Error())
				return
			}
			err = dbErr
		}
	}()

	config := ctx.config.Build
	if len(config.DockerTag) > 0 {
		ctx.imageTag = append(ctx.imageTag, genImageTag(ctx, config.DockerTag))
	}

	buildConf := &image.BuildOption{
		DockerFilePath: config.Dockerfile,
		CtxPath:        ctx.path,
		Tags:           ctx.imageTag,
	}

	output, err := image.Build(ctx, buildConf)
	defer func() {
		err := output.Close()
		if err != nil {
			log.Error("close reader from docker build failed: %s", err.Error())
		}
	}()
	if err != nil {
		return
	}
	bs, err := ioutil.ReadAll(output)
	if err != nil {
		return
	}
	outputLog = string(bs)
	return
}

func genImageTag(ctx *Context, tag string) string {
	_ = ctx.repo.LoadAttributes()
	return fmt.Sprintf("%s/%s/%s:%s", conf.Docker.Registry, ctx.repo.Owner.HarborName, ctx.repo.LowerName, tag)
}
