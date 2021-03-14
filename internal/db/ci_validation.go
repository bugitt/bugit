package db

import (
	"errors"
	"path/filepath"
	"strings"
	"time"
)

type ValidationErrorType int

const (
	ValidationError ValidationErrorType = iota + 1
	ValidationWarn
)

type ValidTaskConfig struct {
	BaseTaskConfig `yaml:",inline"`
	Lang           string   `yaml:"lang"`
	Scope          string   `yaml:"scope"`
	Enable         []string `yaml:"enable"`
	Disable        []string `yaml:"disable"`
	Threshold      int      `yaml:"threshold"`
	Path           string   `yaml:"-"`
}

type ValidationTask struct {
	Issues    []ValidationResult `xorm:"-" json:"-"`
	config    *ValidTaskConfig   `xorm:"-" json:"-"`
	BasicTask `xorm:"extends"`
	BaseModel `xorm:"extends"`
}

type ValidationResult struct {
	ID               int64
	ValidationTaskID int64
	ErrorType        ValidationErrorType
	SourceLog        string
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

func (task *ValidationTask) Run() error {
	config := task.config
	config.Path = filepath.Join(task.pipeTask.Pipeline.CIPath, config.Scope)
	var (
		linter Linter
		err    error
	)
	if len(config.Type) > 0 {
		switch strings.ToLower(config.Type) {
		case "golangci-lint":
			linter, err = golangciLint(config)
			if err != nil {
				return err
			}
		}
	} else {
		switch strings.ToLower(config.Lang) {
		case "go", "golang":
			linter, err = golangciLint(config)
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
	if cnt <= config.Threshold {
		return nil
	}
	return errors.New("to many warnings")
}

func (task *ValidationTask) start() error {
	task.Status = Running
	task.BeginUnix = time.Now().Unix()
	_, err := x.ID(task.ID).Cols("status", "is_started", "begin_unix").Update(task)
	return err
}

func (task *ValidationTask) success() error {
	task.Status = Finished
	task.IsSucceed = true
	task.EndUnix = time.Now().Unix()
	_, err := x.ID(task.ID).Cols("status", "is_succeed", "end_unix").Update(task)
	return err
}

func (task *ValidationTask) failed() error {
	task.Status = Finished
	task.IsSucceed = false
	task.EndUnix = time.Now().Unix()
	_, err := x.ID(task.ID).Cols("status", "is_succeed", "end_unix").Update(task)
	return err
}
