package ci

import (
	"fmt"
	"time"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/conf"
	"git.scs.buaa.edu.cn/iobs/bugit/internal/db"
	"github.com/loheagn/loclo/docker/image"
	log "unknwon.dev/clog/v2"
)

func push(ctx *Context) (err error) {
	err = ctx.updateStage(db.PushStart, -1)
	if err != nil {
		return
	}

	for i, tag := range ctx.imageTag {
		err = pushNum(ctx, i, tag)
		if err != nil {
			return err
		}
	}
	return ctx.updateStage(db.PushEnd, -1)
}

func pushNum(ctx *Context, num int, tag string) (err error) {
	var (
		outputLog string
		begin     = time.Now()
		result    = db.PushResult{
			ImageTag: tag,
			BasicTaskResult: db.BasicTaskResult{
				PipelineID: ctx.pipeline.ID,
				Name:       fmt.Sprintf("push-%d", num+1),
				Describe:   fmt.Sprintf("push image: %s", tag),
			},
		}
	)

	defer func() {
		result.End(begin, err, outputLog)
		dbErr := db.SaveCIResult(result)
		if dbErr != nil {
			log.Error("save push image task %s failed, error message: %s", tag, dbErr.Error())
			if err != nil {
				// 防止dbError将真正的错误覆盖
				return
			}
			err = dbErr
		}
	}()

	outputLog, err = image.Push(ctx, &image.PushOption{
		Tag:      tag,
		Username: conf.Harbor.AdminName,
		Password: conf.Harbor.AdminPasswd,
	})
	if err != nil {
		err = fmt.Errorf("push image %s error: %w", tag, err)
		return
	}

	// 表示本序号的 PreBuild 任务完成
	err = ctx.updateStage(db.PushStart, num+1)
	return
}
