package platform

import (
	"context"
	"encoding/base64"
	"fmt"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/conf"
	json "github.com/json-iterator/go"
	"github.com/rancher/norman/clientbase"
	"github.com/rancher/norman/types"
	clusterClient "github.com/rancher/rancher/pkg/client/generated/cluster/v3"
	managementClient "github.com/rancher/rancher/pkg/client/generated/management/v3"
	projectClient "github.com/rancher/rancher/pkg/client/generated/project/v3"
)

type RancherCli struct {
	globalCliOpt *clientbase.ClientOpts
	cclient      *clusterClient.Client
	mClient      *managementClient.Client
	pClientMap   map[string]*projectClient.Client
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
		pClientMap:   make(map[string]*projectClient.Client),
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

func (cli RancherCli) CreateProject(ctx context.Context, opt *CreateProjectOpt) (*Project, error) {
	limit := &managementClient.ResourceQuotaLimit{
		LimitsCPU:    conf.Deploy.DefaultNSCPULimit,
		LimitsMemory: conf.Deploy.DefaultNSMemLimit,
	}

	// create project
	p, err := cli.mClient.Project.Create(&managementClient.Project{
		Annotations: map[string]string{
			"from": "BuGit",
		},
		ClusterID: conf.Rancher.Cluster,
		ContainerDefaultResourceLimit: &managementClient.ContainerResourceLimit{
			LimitsCPU:    conf.Deploy.DefaultContainerCPULimit,
			LimitsMemory: conf.Deploy.DefaultContainerMemLimit,
		},
		CreatorID:                     conf.Rancher.AdminID,
		Description:                   "Project is created by BuGit.",
		EnableProjectMonitoring:       true,
		Name:                          opt.ProjectName,
		NamespaceDefaultResourceQuota: &managementClient.NamespaceResourceQuota{Limit: limit},
		ResourceQuota:                 &managementClient.ProjectResourceQuota{Limit: limit},
		State:                         "unknown",
	})
	if err != nil {
		return nil, err
	}

	// create namespace
	ns, err := cli.cclient.Namespace.Create(&clusterClient.Namespace{
		CreatorID:   conf.Rancher.AdminID,
		Description: "The namespace is created by BuGit.",
		Name:        opt.ProjectName,
		ProjectID:   p.ID,
	})
	if err != nil {
		return nil, err
	}

	// 为 namespace 创建 secret，以允许其从Harbor中拉取镜像
	pClient := cli.pClientMap[p.ID]
	if pClient == nil {
		pClientOpt := cli.globalCliOpt
		pClientOpt.URL = fmt.Sprintf("%s/projects/%s", cli.globalCliOpt.URL, p.ID)
		pClient, err = projectClient.NewClient(pClientOpt)
		if err != nil {
			return nil, err
		}
		cli.pClientMap[p.ID] = pClient
	}
	_, err = pClient.Secret.Create(&projectClient.Secret{
		Resource: types.Resource{
			Type: "kubernetes.io/dockerconfigjson",
		},
		CreatorID:   conf.Rancher.AdminID,
		Description: "Pull images from Harbor.",
		Immutable:   getBoolPtr(true),
		Name:        ns.Name + "-docker-registry",
		Kind:        "Secret",
		NamespaceId: ns.ID,
		ProjectID:   p.ID,
		StringData:  map[string]string{".dockerconfigjson": encodeDockerRegistryConfig()},
	})
	if err != nil {
		return nil, err
	}

	return &Project{
		Name:     p.Name,
		StringID: p.ID,
	}, nil
}

func encodeDockerRegistryConfig() string {
	type DockerRegistryAuthConfig struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Auth     string `json:"auth"`
	}
	var (
		username = conf.Harbor.AdminName
		password = conf.Harbor.AdminPasswd
	)
	configJson := map[string]map[string]DockerRegistryAuthConfig{
		"auths": {
			conf.Harbor.Host: DockerRegistryAuthConfig{
				Username: username,
				Password: password,
				Auth:     base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password))),
			},
		},
	}
	data, _ := json.Marshal(configJson)
	return string(data)
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
