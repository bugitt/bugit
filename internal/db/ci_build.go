package db

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"

	"gogs.io/gogs/internal/conf"
	log "unknwon.dev/clog/v2"
)

type BuildTaskConfig struct {
	BaseTaskConfig `yaml:",inline"`
	Dockerfile     string `yaml:"dockerfile"`
	Scope          string `yaml:"scope"`
}

type BuildTask struct {
	SourceLog      string `xorm:"TEXT" json:"source_log"`
	ImageTag       string
	BuildErrString string `xorm:"build_err TEXT"`
	BasicTask      `xorm:"extends"`
	BaseModel      `xorm:"extends"`
}

func (task *BuildTask) Run(ctx *CIContext) error {
	config := ctx.config.Build[task.Number-1]
	buildPath := filepath.Join(ctx.path, config.Scope)
	switch strings.ToLower(config.Type) {
	case "docker":
		imageTag := fmt.Sprintf("%s/%s/%s:%s",
			conf.Docker.Registry,
			ctx.owner,
			ctx.repo,
			ctx.commit[:5])

		// Build
		sourceLog, isSuccessful, buildErr, err := BuildImage(config.Dockerfile, buildPath, []string{imageTag})
		if err != nil {
			log.Error(err.Error())
		}
		task.SourceLog = sourceLog
		if !isSuccessful {
			errString, _ := jsoniter.Marshal(buildErr)
			task.BuildErrString = string(errString)
			return errors.New("build image error")
		} else {
			task.BuildErrString = ""
		}
		// 设置镜像tag
		ctx.imageTag = imageTag
	}
	return nil
}

func (task *BuildTask) start() error {
	task.Status = Running
	task.BeginUnix = time.Now().Unix()
	_, err := x.ID(task.ID).Cols("status", "is_started", "begin_unix").Update(task)
	return err
}

func (task *BuildTask) success() error {
	task.Status = Finished
	task.IsSucceed = true
	task.EndUnix = time.Now().Unix()
	_, err := x.ID(task.ID).Cols("status", "is_succeed", "end_unix").Update(task)
	if err != nil {
		return err
	}
	_, err = x.ID(task.ID).Update(task)
	return err
}

func (task *BuildTask) failed() error {
	task.Status = Finished
	task.IsSucceed = false
	task.EndUnix = time.Now().Unix()
	_, err := x.ID(task.ID).Cols("status", "is_succeed", "end_unix").Update(task)
	if err != nil {
		return err
	}
	_, err = x.ID(task.ID).Update(task)
	return err
}
