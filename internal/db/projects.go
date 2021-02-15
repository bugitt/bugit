package db

import "fmt"

type UserProjectOptions struct {
	SenderID int64
	Page     int
	PageSize int
}

type ErrProjectNotExist struct {
	p *Project
}

func (err ErrProjectNotExist) Error() string {
	return fmt.Sprintf("project not exist: %#v", err.p)
}

func IsProjectNotExist(err error) bool {
	_, ok := err.(ErrProjectNotExist)
	return ok
}
