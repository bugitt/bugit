package ci

import (
	"fmt"
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
		err = dockerBuild(ctx, config)
	default:
		err = errors.New("not support build type")
	}
	if err != nil {
		return
	}

	return ctx.updateStage(db.BuildEnd, -1)
}

func dockerBuild(ctx *Context, config *db.BuildTaskConfig) (err error) {
	var (
		outputLog string
		begin     = time.Now()
		result    = db.BuildResult{
			BasicTaskResult: db.BasicTaskResult{
				PipelineID: ctx.pipeline.ID,
				Name:       ctx.config.Build.Name,
				Describe:   ctx.config.Build.Describe,
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

	if len(config.DockerTag) > 0 {
		ctx.imageTag = append(ctx.imageTag, genImageTag(ctx, config.DockerTag))
	}

	buildConf := &image.BuildOption{
		DockerFilePath: config.Dockerfile,
		CtxPath:        ctx.path,
		Tags:           ctx.imageTag,
	}

	outputLog, err = image.Build(ctx, buildConf)
	return
}

func genImageTag(ctx *Context, tag string) string {
	_ = ctx.repo.LoadAttributes()
	return fmt.Sprintf("%s/%s/%s:%s", conf.Docker.Registry, ctx.harborProjectName, ctx.repo.LowerName, tag)
}
