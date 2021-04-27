package db

import "fmt"

type CIError struct {
	Type CIErrType
	err  error
}

func (ciErr *CIError) Error() string {
	return ciErr.err.Error()
}

type CIErrType int

const (
	InternalErrType CIErrType = 500
	UnknownErrType  CIErrType = 501
	TimeoutErrType  CIErrType = 502

	ValidateErrType CIErrType = 100

	BuildErrType CIErrType = 200

	TestErrType CIErrType = 300

	PushErrType CIErrType = 400

	DeployErrType CIErrType = 600
)

type ErrNoNeedDeploy struct {
	Reason string
}

func (err *ErrNoNeedDeploy) Error() string {
	return err.Reason
}

func IsErrNoNeedDeploy(err error) bool {
	_, ok := err.(*ErrNoNeedDeploy)
	return ok
}

type ErrNoValidCIConfig struct {
	RepoName string
	Branch   string
	Commit   string
}

func (err *ErrNoValidCIConfig) Error() string {
	return fmt.Sprintf("can not parse valid CIConfig, repo: %s, branch: %s, commit: %s", err.RepoName, err.Branch, err.Commit)
}

func IsErrNoValidCIConfig(err error) bool {
	_, ok := err.(*ErrNoValidCIConfig)
	return ok
}
