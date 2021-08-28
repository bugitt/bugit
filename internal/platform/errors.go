package platform

type Err string

const (
	ErrProjectNameDuplicate Err = "project name duplicate"
	//ErrUserNameDuplicate Err = "user name duplicate"
)

func (err Err) Error() string {
	return string(err)
}
