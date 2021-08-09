package db

import (
	"errors"
	"path/filepath"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"

	log "unknwon.dev/clog/v2"
)

type TestTaskConfig struct {
	BaseTaskConfig `yaml:",inline"`
	WorkingDir     string `yaml:"workingDir"`
	Cmd            Cmd    `yaml:"cmd"`
}

type BuildTask struct {
	SourceLog       string `xorm:"TEXT" json:"source_log"`
	ImageTag        string
	BuildErrString  string `xorm:"build_err TEXT"`
	BasicTaskResult `xorm:"extends"`
	BaseModel       `xorm:"extends"`
}

func (task *BuildTask) Run(ctx *CIContext) error {
	config := ctx.config.Build
	buildPath := filepath.Join(ctx.path, config.Scope)
	switch strings.ToLower(config.Type) {
	case "docker":
		// Build
		sourceLog, isSuccessful, buildErr, err := BuildImage(config.Dockerfile, buildPath, []string{ctx.imageTag})
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
	task.IsSuccessful = true
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
	task.IsSuccessful = false
	task.EndUnix = time.Now().Unix()
	_, err := x.ID(task.ID).Cols("status", "is_succeed", "end_unix").Update(task)
	if err != nil {
		return err
	}
	_, err = x.ID(task.ID).Update(task)
	return err
}
