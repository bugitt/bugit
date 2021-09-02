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

var cliSet []Actor

// Init 初始化各个平台的客户端
func Init() error {
	harborCli, err := getHarborClient()
	if err != nil {
		return err
	}
	cliSet = append(cliSet, harborCli)

	return nil
}

func CreateUser(ctx context.Context, opt *CreateUserOpt) (*User, error) {
	var user *User
	for _, cli := range cliSet {
		// 先创建用户本身
		u, err := cli.CreateUser(ctx, opt)
		if err != nil {
			return nil, err
		}
		user = u

		// 然后创建用户的个人项目
		p, err := cli.CreateProject(ctx, &CreateProject{ProjectName: u.Name})
		if err != nil {
			return nil, err
		}

		// 然后将这个用户设置为自己个人项目的管理员
		if err = cli.AddAdmin(ctx, u, p); err != nil {
			return nil, err
		}
	}

	return user, nil
}
