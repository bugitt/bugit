package platform

import (
	"context"
	"fmt"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/conf"
	"github.com/rancher/norman/clientbase"
	clusterClient "github.com/rancher/rancher/pkg/client/generated/cluster/v3"
	managementClient "github.com/rancher/rancher/pkg/client/generated/management/v3"
)

type RancherCli struct {
	globalCliOpt *clientbase.ClientOpts
	cclient      *clusterClient.Client
	mClient      *managementClient.Client
}

func NewRancherCli() (*RancherCli, error) {
	opt := &clientbase.ClientOpts{
		URL:       conf.Rancher.Url,
		AccessKey: conf.Rancher.AccessKey,
		SecretKey: conf.Rancher.SecretKey,
		TokenKey:  conf.Rancher.Token,
	}
	mClient, err := managementClient.NewClient(opt)
	if err != nil {
		return nil, err
	}

	copt := opt
	copt.URL = fmt.Sprintf("%s/clusters/%s", opt.URL, conf.Rancher.Cluster)
	cc, err := clusterClient.NewClient(copt)
	return &RancherCli{
		globalCliOpt: opt,
		cclient:      cc,
		mClient:      mClient,
	}, nil
}

func (cli RancherCli) CreateUser(ctx context.Context, opt *CreateUserOpt) (*User, error) {
	mc := cli.mClient

	// 创建用户
	u, err := mc.User.Create(&managementClient.User{
		Annotations: map[string]string{
			"from": "BuGit",
		},
		Description:        "User is created by BuGit.",
		Enabled:            getBoolPtr(true),
		MustChangePassword: true,
		Name:               opt.RealName,
		Password:           conf.Harbor.DefaultPasswd,
		State:              "unknown",
		Username:           opt.StudentID,
		CreatorID:          conf.Rancher.AdminID,
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
