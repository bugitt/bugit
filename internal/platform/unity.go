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

var (
	harborCli *HarborCli
	cliSet    []Actor
)

// Init 初始化各个平台的客户端
func Init() (err error) {
	harborCli, err = getHarborClient()
	if err != nil {
		return err
	}
	cliSet = append(cliSet, harborCli)

	return nil
}

func CreateHarborUser(ctx context.Context, studentID, userName, email, realName string) (userID int64, err error) {
	u, err := createUser(ctx, harborCli, &CreateUserOpt{
		StudentID: studentID,
		UserName:  userName,
		Email:     email,
		RealName:  realName,
	})
	if err != nil {
		return
	}
	return u.ID, err
}

func CreateHarborProject(ctx context.Context, userID int64, projectName string) (projectID int64, err error) {
	p, err := createProject(ctx, harborCli, &User{ID: userID}, projectName)
	if err != nil {
		return 0, err
	}
	return p.ID, err
}

func createUser(ctx context.Context, cli Actor, opt *CreateUserOpt) (*User, error) {
	// 先创建用户本身
	u, err := cli.CreateUser(ctx, opt)
	if err != nil {
		return nil, err
	}

	// 然后创建用户的个人项目
	p, err := cli.CreateProject(ctx, &CreateProject{ProjectName: u.Name})
	if err != nil {
		return nil, err
	}

	// 然后将这个用户设置为自己个人项目的管理员
	if err = cli.AddAdmin(ctx, u, p); err != nil {
		return nil, err
	}

	return u, err
}

func createProject(ctx context.Context, cli Actor, u *User, projectName string) (p *Project, err error) {
	p, err = cli.CreateProject(ctx, &CreateProject{ProjectName: projectName})
	if err != nil {
		return
	}

	err = cli.AddAdmin(ctx, u, p)
	if err != nil {
		return
	}
	return p, err
}
