package platform

import "context"

type CreateUserOpt struct {
	StudentID string
	UserName  string
	Email     string
	RealName  string
}

type CreateProject struct {
	ProjectName string
	OwnerName   string
}

type Actor interface {
	CreateUser(context.Context, *CreateUserOpt) (*User, error)
	CreateProject(context.Context, *CreateProject) (*Project, error)
	AddAdmin(context.Context, *User, *Project) error
}

type Project struct {
	Name string
	ID   int64
}

type User struct {
	Name string
	ID   int64
}

var (
	harborCli *HarborCli
)

// Init 初始化各个平台的客户端
func Init() (err error) {
	harborCli, err = getHarborClient()
	if err != nil {
		return err
	}
	return nil
}
