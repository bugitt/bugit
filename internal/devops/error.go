package devops

import "fmt"

type ErrConfFileNotFound struct {
	repoPath string
}

func (err ErrConfFileNotFound) Error() string {
	return fmt.Sprintf("devops conf file not found: repo: %s", err.repoPath)
}

func IsErrConfFileNotFound(err error) bool {
	_, ok := err.(ErrConfFileNotFound)
	return ok
}
