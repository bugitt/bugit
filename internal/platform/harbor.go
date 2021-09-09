package platform

import (
	"context"
	"strconv"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/conf"
	"github.com/mittwald/goharbor-client/v4/apiv2"
	modelv2 "github.com/mittwald/goharbor-client/v4/apiv2/model"
	legacymodel "github.com/mittwald/goharbor-client/v4/apiv2/model/legacy"
	"github.com/mittwald/goharbor-client/v4/apiv2/project"
	"github.com/mittwald/goharbor-client/v4/apiv2/user"
)

type HarborCli struct {
	*apiv2.RESTClient
}

func getHarborClient() (*HarborCli, error) {
	cli, err := apiv2.NewRESTClientForHost(conf.Harbor.Url, conf.Harbor.AdminName, conf.Harbor.AdminPasswd)
	if err != nil {
		return nil, err
	}
	return &HarborCli{cli}, nil
}

func (cli HarborCli) CreateUser(ctx context.Context, opt *CreateUserOpt) (*User, error) {
	userName := opt.StudentID
	// check whether user already exists
	u, err := cli.GetUser(ctx, userName)
	if err != nil {
		if _, ok := err.(*user.ErrUserNotFound); ok {
			// 如果用户不存在，则创建用户
			u, err = cli.NewUser(ctx, userName, opt.Email, opt.RealName, conf.Harbor.DefaultPasswd, "User created by BuGit")
			if err != nil {
				return nil, err
			}
		}
	}
	return &User{Name: u.Username, IntID: u.UserID}, nil
}

func (cli *HarborCli) getHarborUP(ctx context.Context, userID, projectID int64) (*legacymodel.User, *modelv2.Project, error) {
	u, err := cli.GetUserByID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	p, err := cli.GetProject(ctx, strconv.FormatInt(projectID, 10))
	return u, p, err
}

func (cli HarborCli) CreateProject(ctx context.Context, opt *CreateProject) (*Project, error) {
	projectName := PrettyName(opt.ProjectName)

	// check whether project already exists
	// if yes, return error
	_, err := cli.GetProject(ctx, projectName)
	if err != nil {
		if _, ok := err.(*project.ErrProjectNotFound); !ok {
			// 如果是异常错误，而不是没找到
			return nil, err
		}
	} else {
		// 否则的话，说明项目名称重复了，提醒用户该换名字了
		return nil, ErrProjectNameDuplicate
	}

	p, err := cli.NewProject(ctx, projectName, getInt64Ptr(-1))
	if err != nil {
		return nil, err
	}

	// The role id 1 for projectAdmin, 2 for developer, 3 for guest, 4 for maintainer
	return &Project{Name: projectName, IntID: int64(p.ProjectID)}, nil
}

func (cli HarborCli) AddOwner(ctx context.Context, u *User, p *Project) error {
	return cli.addMember(ctx, u, p, 1)
}

func (cli HarborCli) addMember(ctx context.Context, u *User, p *Project, roleID int) error {
	nu, np, err := cli.getHarborUP(ctx, u.IntID, p.IntID)
	if err != nil {
		return err
	}
	return cli.AddProjectMember(ctx, np, nu, roleID)
}

func (cli HarborCli) RemoveMember(ctx context.Context, u *User, p *Project) error {
	nu, np, err := cli.getHarborUP(ctx, u.IntID, p.IntID)
	if err != nil {
		return err
	}
	return cli.DeleteProjectMember(ctx, np, nu)
}

func (cli HarborCli) DeleteProject(ctx context.Context, p *Project) error {
	harborProject, err := cli.GetProject(ctx, strconv.FormatInt(p.IntID, 10))
	if err != nil {
		return err
	}
	return cli.RESTClient.DeleteProject(ctx, harborProject)
}

//func DeleteProject(ctx context.Context, projectID string) error {
//	// FIXME: 补充删除项目中的repo的api后，再启用删除Harbor中项目的逻辑
//	// client, err := getHarborClient()
//	// if err != nil {
//	// 	return err
//	// }
//	// p, err := client.GetProject(ctx, projectID)
//	// if err != nil {
//	// 	return err
//	// }
//	// return client.DeleteProject(ctx, p)
//	return nil
//}
