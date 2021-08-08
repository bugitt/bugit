package db

import (
	"errors"
	"path/filepath"
	"strings"
	"time"
)

type ValidationErrorType1 int

const (
	ValidationError ValidationErrorType1 = iota + 1
	ValidationWarn
)

type ValidTaskConfig struct {
	BaseTaskConfig `yaml:",inline"`
	Lang           string `yaml:"lang"`
	Scope          string `yaml:"scope"`
	WorkDir        string `yaml:"workDir"`
	Skip           bool   `yaml:"skip"`
	Path           string `yaml:"-"`
}

type ValidationResult struct {
	ID               int64
	ValidationTaskID int64
	ErrorType        ValidationErrorType1
	SourceLog        string `xorm:"TEXT"`
	FromLinter       string
	Text             string
	SourceLines      []string
	Pos              ValidationResultPos `xorm:"-"`
	PosString        string
}

type ValidationResultPos struct {
	FileName string
	Offset   int
	Line     int
	Column   int
}

type Linter interface {
	ConvertValidationResult(taskID int64) []*ValidationResult
}

func (task *PreBuildResult) Run(ctx *CIContext) error {
	config := ctx.config.Validate[task.Number-1]
	config.Path = filepath.Join(ctx.path, config.Scope)
	var (
		linter Linter
		err    error
	)
	if len(config.Type) > 0 {
		switch strings.ToLower(config.Type) {
		case "golangci-lint":
			linter, err = golangciLint(&config)
			if err != nil {
				return err
			}
		}
	} else {
		switch strings.ToLower(config.Lang) {
		case "go", "golang":
			linter, err = golangciLint(&config)
			if err != nil {
				return err
			}
		}
	}

	resultList := linter.ConvertValidationResult(task.ID)
	var cnt int
	for _, result := range resultList {
		_, err := x.Insert(result)
		if err != nil {
			return err
		}
		if result.ErrorType == ValidationError {
			return errors.New("ci lint: code have errors")
		}
		cnt++
	}
	// if cnt <= config.Threshold {
	// 	return nil
	// }
	return errors.New("to many warnings")
}

func (task *PreBuildResult) start() error {
	task.Status = Running
	task.BeginUnix = time.Now().Unix()
	_, err := x.ID(task.ID).Cols("status", "is_started", "begin_unix").Update(task)
	return err
}

func (task *PreBuildResult) success() error {
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

func (task *PreBuildResult) failed() error {
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
