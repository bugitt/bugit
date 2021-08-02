package ci

import (
	"context"
	"fmt"
	"time"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/db"
	log "unknwon.dev/clog/v2"
)

// run CI CI过程中生成的error应该被自己消费掉
func run(pipeline *db.Pipeline) {
	// 一个task最多只允许跑一小时
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()

	var err error

	defer func() {
		if err != nil {
			log.Error("pipe CI error: %s", err.Error())
			if err = pipeline.Fail(err); err != nil {
				log.Error("update pipeline error: %s", err.Error())
			}
		}
	}()

	// work
	done := make(chan error)
	go func() {
		done <- func() (err error) {
			defer func() {
				if panicErr := recover(); panicErr != nil {
					err = fmt.Errorf("panic occurred: %#v", panicErr)
				}
			}()
			ctx, err := prepareCtx(pipeline)
			if err != nil {
				return err
			}
			return contextCI(ctx)
		}()
	}()
	select {
	case err = <-done:
	case <-ctx.Done():
		err = ctx.Err()
	}

	// 保证打上结束的时间戳
	if err == nil {
		log.Info("pipe CI success: %d", pipeline.ID)
		err = pipeline.Succeed()
	}
}
