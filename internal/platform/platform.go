package platform

import (
	"context"
	"strings"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/conf"
)

type CreateUserOpt struct {
	StudentID  string
	Email      string
	RealName   string
	Password   string
	NeedPrefix bool
}

type Actor interface {
	CreateUser(context.Context, *CreateUserOpt) (*User, error)
	GetUser(context.Context, string) (*User, error)
	GetProject(context.Context, string) (*Project, error)
	CreateProject(context.Context, string) (*Project, error)
	DeleteProject(context.Context, *Project) error
	CheckOwner(context.Context, *User, string) (bool, error)
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
	harborCli *HarborCli
	//rancherCli *RancherCli
	ksCli  *KSCli
	cliSet []Actor
)

// Init 初始化各个平台的客户端
func Init() {
	//var err error
	harborCli = NewHarborCli(conf.Harbor.Host, conf.Harbor.AdminName, conf.Harbor.AdminPasswd)
	cliSet = append(cliSet, harborCli)

	//rancherCli, err = NewRancherCli()
	//if err != nil {
	//	panic(err)
	//}
	//cliSet = append(cliSet, rancherCli)

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

func CreateHarborUser(ctx context.Context, studentID, email, realName, password string) (userID, projectID int64, err error) {
	u, p, err := createUser(ctx, harborCli, &CreateUserOpt{
		StudentID: studentID,
		Email:     email,
		RealName:  realName,
		Password:  password,
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

//func CreateRancherUser(studentID, realName string) (userID string, err error) {
//	u, err := rancherCli.CreateUser(context.Background(), &CreateUserOpt{
//		StudentID: studentID,
//		RealName:  realName,
//	})
//	if err != nil {
//		return "", err
//	}
//	return u.StringID, nil
//}

//func CreateRancherProject(userID, projectName string) (projectID string, err error) {
//	p, err := createProject(context.Background(), harborCli, &User{StringID: userID}, projectName)
//	if err != nil {
//		return "", err
//	}
//	return p.StringID, err
//}
//
//func AddRancherOwner(userID, projectID string) (err error) {
//	return addOwner(context.Background(), harborCli, &User{StringID: userID}, &Project{StringID: projectID})
//}
//
//func DeleteRancherProject(projectID string) (err error) {
//	return deleteProject(context.Background(), harborCli, &Project{StringID: projectID})
//}
//
//func RemoveRancherProjectMember(userID, projectID string) (err error) {
//	return removeMember(context.Background(), harborCli, &User{StringID: userID}, &Project{StringID: projectID})
//}

func CreateKSUser(studentID, email, password string) (userName, projectName string, err error) {
	studentID = strings.ToLower(studentID)
	u, p, err := createUser(context.Background(), ksCli, &CreateUserOpt{
		StudentID:  studentID,
		Email:      email,
		RealName:   studentID,
		Password:   password,
		NeedPrefix: true,
	})
	if err != nil {
		return
	}
	return u.Name, p.Name, nil
}

func CreateKSProject(username string, projectName string) (projectID string, err error) {
	username = strings.ToLower(username)
	projectName = strings.ToLower(projectName)
	p, err := createProject(context.Background(), ksCli, &User{Name: username}, projectName)
	if err != nil {
		return "", err
	}
	return p.Name, err
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

func createUser(ctx context.Context, cli Actor, opt *CreateUserOpt) (u *User, p *Project, err error) {
	prettyCreateUserOpt(opt)
	// 先创建用户本身
	u, _ = cli.GetUser(ctx, opt.StudentID)
	if u == nil {
		u, err = cli.CreateUser(ctx, opt)
		if err != nil {
			return nil, nil, err
		}
	}

	// 然后创建用户的个人项目
	p, err = createProject(ctx, cli, u, getProjectName(opt))
	if err != nil {
		return nil, nil, err
	}

	return u, p, nil
}

func createProject(ctx context.Context, cli Actor, u *User, projectName string) (p *Project, err error) {
	p, _ = cli.GetProject(ctx, projectName)
	if p == nil {
		p, err = cli.CreateProject(ctx, projectName)
		if err != nil {
			return
		}
	}

	ok, _ := cli.CheckOwner(ctx, u, projectName)
	if !ok {
		err = cli.AddOwner(ctx, u, p)
		if err != nil {
			return
		}
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

func prettyCreateUserOpt(opt *CreateUserOpt) {
	if len(opt.Password) <= 0 {
		opt.Password = conf.Harbor.DefaultPasswd
	}
	opt.StudentID = strings.ToLower(opt.StudentID)
	if len(opt.RealName) <= 0 {
		opt.RealName = opt.StudentID
	}
}

func getProjectName(opt *CreateUserOpt) string {
	if opt.NeedPrefix {
		return "project-" + strings.ToLower(opt.StudentID)
	} else {
		return strings.ToLower(opt.StudentID)
	}
}
