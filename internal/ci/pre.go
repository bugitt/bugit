package ci

import (
	"fmt"
	"time"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/db"
	"github.com/loheagn/loclo/docker/container"
	log "unknwon.dev/clog/v2"
)

func preBuild(ctx *Context) (err error) {
	err = ctx.updateStage(db.PreBuildStart, -1)
	if err != nil {
		return
	}

	for i, preConf := range ctx.config.PreBuild {
		err = preBuildNum(ctx, i)
		if err != nil && !preConf.CanSkip {
			return err
		}
	}

	return ctx.updateStage(db.PreBuildEnd, -1)
}

func preBuildNum(ctx *Context, num int) (err error) {
	var (
		outputLog string
		begin     = time.Now()
	)

	defer func() {
		result := db.PreBuildResult{
			Number: num + 1,
			BasicTask: db.BasicTask{
				PipelineID: ctx.pipeline.ID,
				Log:        outputLog,
				Duration:   time.Since(begin).Milliseconds(),
				BeginUnix:  begin.Unix(),
				EndUnix:    time.Now().Unix(),
			},
		}
		if err != nil {
			result.ErrMsg = err.Error()
		} else {
			result.IsSuccessful = true
		}
		dbErr := db.SaveCIResult(result)
		if dbErr != nil {
			if err != nil {
				log.Error("save pre build result %d failed, error message: %s", num, dbErr.Error())
				return
			}
			err = dbErr
		}
	}()

	preConf := ctx.config.PreBuild[num]
	runConf := preConf.ToRunConf(ctx.path, preConf.Image)

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
