package ci

import (
	"fmt"
	"time"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/db"
	"github.com/loheagn/loclo/docker/container"
	log "unknwon.dev/clog/v2"
)

func postBuild(ctx *Context) (err error) {
	err = ctx.updateStage(db.PostBuildStart, -1)
	if err != nil {
		return
	}

	for i, postConf := range ctx.config.PostBuild {
		err = postBuildNum(ctx, i)
		if err != nil && !postConf.CanSkip {
			return err
		}
	}

	return ctx.updateStage(db.PostBuildEnd, -1)
}

func postBuildNum(ctx *Context, num int) (err error) {
	var (
		outputLog string
		begin     = time.Now()
		result    = db.PostBuildResult{
			Number: num + 1,
			BasicTaskResult: db.BasicTaskResult{
				PipelineID: ctx.pipeline.ID,
				Name:       ctx.config.PostBuild[num].Name,
				Describe:   ctx.config.PostBuild[num].Describe,
			},
		}
	)

	defer func() {
		result.End(begin, err, outputLog)
		dbErr := db.SaveCIResult(result)
		if dbErr != nil {
			if err != nil {
				log.Error("save post build result %d failed, error message: %s", num, dbErr.Error())
				return
			}
			err = dbErr
		}
	}()

	postConf := ctx.config.PostBuild[num]
	runConf := postConf.ToRunConf(ctx.path, ctx.imageTag[0])

	outputLog, exitCode, err := container.Run(ctx, runConf)
	if err != nil {
		err = fmt.Errorf("run container error: %w", err)
		return
	}
	if exitCode != 0 {
		err = fmt.Errorf("container exit abnormally: %d", exitCode)
		return
	}

	// 表示本序号的 PreBuild 任务完成
	err = ctx.updateStage(db.PreBuildStart, num+1)
	return
}
