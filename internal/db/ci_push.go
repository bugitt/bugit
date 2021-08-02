package db

import (
	"errors"
	"time"

	jsoniter "github.com/json-iterator/go"
)

type PushTask struct {
	SourceLog     string `xorm:"TEXT" json:"source_log"`
	ImageTag      string
	PushErrString string `xorm:"push_err TEXT"`
	BasicTask     `xorm:"extends"`
	BaseModel     `xorm:"extends"`
}

func (task *PushTask) Run(ctx *CIContext) error {
	// Push
	sourceLog, isSuccessful, buildErr, err := PushImage(ctx.imageTag)
	if err != nil {
		return err
	}
	task.SourceLog = sourceLog
	if !isSuccessful {
		errString, _ := jsoniter.Marshal(buildErr)
		task.PushErrString = string(errString)
		return errors.New("push image error")
	} else {
		task.PushErrString = ""
	}
	return nil
}

func (task *PushTask) start() error {
	task.Status = Running
	task.BeginUnix = time.Now().Unix()
	_, err := x.ID(task.ID).Cols("status", "is_started", "begin_unix").Update(task)
	return err
}

func (task *PushTask) success() error {
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

func (task *PushTask) failed() error {
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
