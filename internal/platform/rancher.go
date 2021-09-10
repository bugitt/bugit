package platform

import (
	"context"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/conf"
	managementClient "github.com/rancher/rancher/pkg/client/generated/management/v3"
)

type RancherCli struct {
	mClient *managementClient.Client
}

func NewRancherCli() (*RancherCli, error) {
	// TODO: 完善 OPT
	mClient, err := managementClient.NewClient(nil)
	if err != nil {
		return nil, err
	}
	return &RancherCli{mClient: mClient}, nil
}

func (cli RancherCli) CreateUser(ctx context.Context, opt *CreateUserOpt) (*User, error) {
	mc := cli.mClient

	// 创建用户
	u, err := mc.User.Create(&managementClient.User{
		Description:        "User is created by BuGit.",
		Enabled:            getBoolPtr(true),
		MustChangePassword: true,
		Name:               opt.RealName,
		Password:           conf.Harbor.DefaultPasswd,
		State:              "unknown",
		Username:           opt.StudentID,
	})
	if err != nil {
		return nil, err
	}

	// 创建user角色
	if err := cli.createUserRole(u.ID, "user"); err != nil {
		return nil, err
	}

	// 允许用户创建新集群
	if err := cli.createUserRole(u.ID, "clusters-create"); err != nil {
		return nil, err
	}

	return &User{
		Name:     u.Name,
		StringID: u.ID,
	}, nil
}

func (cli RancherCli) createUserRole(userID, roleID string) error {
	_, err := cli.mClient.GlobalRoleBinding.Create(&managementClient.GlobalRoleBinding{
		GlobalRoleID: roleID,
		UserID:       userID,
	})
	return err
}

func (cli RancherCli) CreateProject(ctx context.Context, project *CreateProject) (*Project, error) {
	panic("implement me")
}

func (cli RancherCli) DeleteProject(ctx context.Context, project *Project) error {
	panic("implement me")
}

func (cli RancherCli) AddOwner(ctx context.Context, user *User, project *Project) error {
	panic("implement me")
}

func (cli RancherCli) RemoveMember(ctx context.Context, u *User, p *Project) error {
	panic("implement me")
}
