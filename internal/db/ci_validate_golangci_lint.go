package db

import (
	"bytes"
	"fmt"
	"os/exec"

	jsoniter "github.com/json-iterator/go"
)

type GolangciResult struct {
	Issues   []GolangciIssues `json:"Issues"`
	ErrorLog string           `json:"-"`
	Report   GolangciReport   `json:"Report"`
}
type GolangciPos struct {
	Filename string `json:"Filename"`
	Offset   int    `json:"Offset"`
	Line     int    `json:"Line"`
	Column   int    `json:"Column"`
}
type GolangciIssues struct {
	FromLinter           string      `json:"FromLinter"`
	Text                 string      `json:"Text"`
	Severity             string      `json:"Severity"`
	SourceLines          []string    `json:"SourceLines"`
	Replacement          interface{} `json:"Replacement"`
	Pos                  GolangciPos `json:"Pos"`
	ExpectNoLint         bool        `json:"ExpectNoLint"`
	ExpectedNoLintLinter string      `json:"ExpectedNoLintLinter"`
}
type GolangciLinters struct {
	Name             string `json:"Name"`
	Enabled          bool   `json:"Enabled,omitempty"`
	EnabledByDefault bool   `json:"EnabledByDefault,omitempty"`
}
type GolangciReport struct {
	Linters []GolangciLinters `json:"Linters"`
}

func golangciLint(config *ValidTaskConfig) (Linter, error) {
	args := []string{
		"run",
		"--out-format",
		"json",
	}
	if len(config.Disable) > 0 {
		args = append(args, "--disable")
		args = append(args, config.Disable...)
	}
	if len(config.Enable) > 0 {
		args = append(args, "--enable")
		args = append(args, config.Enable...)
	}
	cmd := exec.Command("golangci-lint", args...)
	cmd.Dir = config.Path
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code := exitErr.ExitCode()
			if code != 1 {
				return &GolangciResult{
					ErrorLog: output.String(),
				}, nil
			}
		} else {
			return nil, err
		}
	}
	result := GolangciResult{}
	if err := jsoniter.Unmarshal(output.Bytes(), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (result *GolangciResult) ConvertValidationResult(taskID int64) []*ValidationResult {
	if len(result.ErrorLog) > 0 {
		return []*ValidationResult{{ErrorType: ValidationError, SourceLog: result.ErrorLog}}
	}
	n := len(result.Issues)
	valResultList := make([]*ValidationResult, 0, n)
	for _, issue := range result.Issues {
		valResultList = append(valResultList, &ValidationResult{
			ValidationTaskID: taskID,
			ErrorType:        ValidationWarn,
			FromLinter:       issue.FromLinter,
			Text:             issue.Text,
			SourceLines:      issue.SourceLines,
			Pos: ValidationResultPos{
				FileName: issue.Pos.Filename,
				Offset:   issue.Pos.Offset,
				Line:     issue.Pos.Line,
				Column:   issue.Pos.Column,
			},
			PosString: fmt.Sprintf("%s:%d:%d", issue.Pos.Filename, issue.Pos.Line, issue.Pos.Column),
		})
	}
	return valResultList
}
