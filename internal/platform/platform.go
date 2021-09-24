package platform

import (
	"context"
	"strings"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/conf"
)

type CreateUserOpt struct {
	StudentID string
	UserName  string
	Email     string
	RealName  string
	Password  string
}

type CreateProjectOpt struct {
	ProjectName string
}

type Actor interface {
	CreateUser(context.Context, *CreateUserOpt) (*User, error)
	CreateProject(context.Context, *CreateProjectOpt) (*Project, error)
	DeleteProject(context.Context, *Project) error
	AddOwner(context.Context, *User, *Project) error
	RemoveMember(ctx context.Context, u *User, p *Project) error
}

type Project struct {
	Name     string
	IntID    int64
	StringID string
}

type User struct {
	Name     string
	IntID    int64
	StringID string
}

var (
	harborCli  *HarborCli
	rancherCli *RancherCli
	ksCli      *KSCli
	cliSet     []Actor
)

// Init 初始化各个平台的客户端
func Init() {
	var err error
	harborCli, err = getHarborClient()
	if err != nil {
		panic(err)
	}
	cliSet = append(cliSet, harborCli)

	rancherCli, err = NewRancherCli()
	if err != nil {
		panic(err)
	}
	cliSet = append(cliSet, rancherCli)

	ksCli = NewKSCli(
		conf.KS.KubernetesURL,
		conf.KS.KSAdmin,
		conf.KS.KSPassword,
		conf.Harbor.Host,
		conf.Harbor.AdminName,
		conf.Harbor.AdminPasswd,
	)
	cliSet = append(cliSet, ksCli)
}

func CreateHarborUser(ctx context.Context, studentID, userName, email, realName string) (userID, projectID int64, err error) {
	u, p, err := createUser(ctx, harborCli, &CreateUserOpt{
		StudentID: studentID,
		UserName:  userName,
		Email:     email,
		RealName:  realName,
	})
	if err != nil {
		return
	}
	return u.IntID, p.IntID, nil
}

func CreateHarborProject(ctx context.Context, userID int64, projectName string) (projectID int64, err error) {
	p, err := createProject(ctx, harborCli, &User{IntID: userID}, projectName)
	if err != nil {
		return 0, err
	}
	return p.IntID, err
}

func AddHarborOwner(ctx context.Context, userID, projectID int64) (err error) {
	return addOwner(ctx, harborCli, &User{IntID: userID}, &Project{IntID: projectID})
}

func DeleteHarborProject(ctx context.Context, projectID int64) (err error) {
	return deleteProject(ctx, harborCli, &Project{IntID: projectID})
}

func RemoveHarborProjectMember(ctx context.Context, userID, projectID int64) (err error) {
	return removeMember(ctx, harborCli, &User{IntID: userID}, &Project{IntID: projectID})
}

func GetHarborProjectName(ctx context.Context, projectID int64) (name string, err error) {
	harborP, err := harborCli.GetProject(ctx, int2Str(projectID))
	if err != nil {
		return "", err
	}
	return harborP.Name, nil
}

func CreateRancherUser(studentID, realName string) (userID, projectID string, err error) {
	u, p, err := createUser(context.Background(), rancherCli, &CreateUserOpt{
		StudentID: studentID,
		RealName:  realName,
	})
	if err != nil {
		return
	}
	return u.StringID, p.StringID, nil
}

func CreateRancherProject(userID, projectName string) (projectID string, err error) {
	p, err := createProject(context.Background(), harborCli, &User{StringID: userID}, projectName)
	if err != nil {
		return "", err
	}
	return p.StringID, err
}

func AddRancherOwner(userID, projectID string) (err error) {
	return addOwner(context.Background(), harborCli, &User{StringID: userID}, &Project{StringID: projectID})
}

func DeleteRancherProject(projectID string) (err error) {
	return deleteProject(context.Background(), harborCli, &Project{StringID: projectID})
}

func RemoveRancherProjectMember(userID, projectID string) (err error) {
	return removeMember(context.Background(), harborCli, &User{StringID: userID}, &Project{StringID: projectID})
}

func CreateKSUser(ctx context.Context, studentID, email string) (userName, projectName string, err error) {
	studentID = strings.ToLower(studentID)
	u, p, err := createUser(ctx, ksCli, &CreateUserOpt{
		StudentID: studentID,
		UserName:  studentID,
		Email:     email,
		RealName:  studentID,
	})
	if err != nil {
		return
	}
	return u.Name, p.Name, nil
}

func CreateKSProject(ctx context.Context, username string, projectName string) (projectID int64, err error) {
	username = strings.ToLower(username)
	projectName = strings.ToLower(projectName)
	p, err := createProject(ctx, ksCli, &User{Name: username}, projectName)
	if err != nil {
		return 0, err
	}
	return p.IntID, err
}

func AddKSOwner(username, projectName string) (err error) {
	return addOwner(context.Background(), ksCli, &User{Name: username}, &Project{Name: projectName})
}

func DeleteKSProject(projectName string) (err error) {
	return deleteProject(context.Background(), ksCli, &Project{Name: projectName})
}

func RemoveKSProjectMember(username, projectName string) (err error) {
	return removeMember(context.Background(), ksCli, &User{Name: username}, &Project{Name: projectName})
}

func createUser(ctx context.Context, cli Actor, opt *CreateUserOpt) (*User, *Project, error) {
	// 先创建用户本身
	u, err := cli.CreateUser(ctx, opt)
	if err != nil {
		return nil, nil, err
	}

	// 然后创建用户的个人项目
	p, err := cli.CreateProject(ctx, &CreateProjectOpt{ProjectName: u.Name})
	if err != nil {
		return nil, nil, err
	}

	// 然后将这个用户设置为自己个人项目的管理员
	if err = cli.AddOwner(ctx, u, p); err != nil {
		return nil, nil, err
	}

	return u, p, nil
}

func createProject(ctx context.Context, cli Actor, u *User, projectName string) (p *Project, err error) {
	p, err = cli.CreateProject(ctx, &CreateProjectOpt{ProjectName: projectName})
	if err != nil {
		return
	}

	err = cli.AddOwner(ctx, u, p)
	if err != nil {
		return
	}
	return p, err
}

func addOwner(ctx context.Context, cli Actor, u *User, p *Project) (err error) {
	return cli.AddOwner(ctx, u, p)
}

func deleteProject(ctx context.Context, cli Actor, p *Project) (err error) {
	return cli.DeleteProject(ctx, p)
}

func removeMember(ctx context.Context, cli Actor, u *User, p *Project) (err error) {
	return cli.RemoveMember(ctx, u, p)
}
