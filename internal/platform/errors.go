package platform

type Err string

const (
	ErrDuplicate Err = "project name duplicate"
	ErrNotFound  Err = "resource not found"
)

func (err Err) Error() string {
	return string(err)
}
