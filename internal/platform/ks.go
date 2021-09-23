package platform

import (
	"context"
	"strings"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/conf"
	"git.scs.buaa.edu.cn/iobs/bugit/internal/kube"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"kubesphere.io/api/constants"
	iam "kubesphere.io/api/iam/v1alpha2"
	tenant "kubesphere.io/api/tenant/v1alpha1"
	"kubesphere.io/client-go/client"
	"kubesphere.io/client-go/client/generic"
)

type KSCli struct {
	client.Client
	username  string
	password  string
	url       string
	harborOpt *kube.HarborOpt
}

var (
	_ = &client.URLOptions{
		Group:   "iam.kubesphere.io",
		Version: "v1alpha2",
	}

	_ = &client.URLOptions{
		Group:   "tenant.kubesphere.io",
		Version: "v1alpha2",
	}

	AdminCreatorAnnotation = map[string]string{
		constants.CreatorAnnotationKey: AdminName,
	}
)

const (
	MainWorkspace             = "main-workspace"
	MainWorkspaceViewer       = "main-workspace-viewer"
	ApiGroupIAM               = "iam.kubesphere.io"
	ApiGroupRBACAuthorization = "rbac.authorization.k8s.io"
	KindUser                  = "User"
	AdminName                 = "admin"
)

func NewKSCli(url, adminName, adminPassword, harborHost, harborAdminName, harborAdminPassword string) *KSCli {
	config := &rest.Config{
		Host:     url,
		Username: adminName,
		Password: adminPassword,
	}
	if err := iam.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		panic(err)
	}
	cli := generic.NewForConfigOrDie(config, client.Options{Scheme: scheme.Scheme})
	return &KSCli{
		Client:   cli,
		username: adminName,
		password: adminPassword,
		url:      url,
		harborOpt: &kube.HarborOpt{
			Username: harborAdminName,
			Password: harborAdminPassword,
			Host:     harborHost,
		}}
}

func (cli KSCli) CreateUser(ctx context.Context, opt *CreateUserOpt) (*User, error) {
	password := opt.Password
	if len(password) <= 0 {
		password = conf.Harbor.DefaultPasswd
	}
	u := &iam.User{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				iam.GlobalRoleAnnotation: "platform-regular",
			},
			Name: opt.StudentID,
		},
		Spec: iam.UserSpec{
			Description:       "The user is created by BuGit.",
			DisplayName:       opt.StudentID,
			Email:             opt.Email,
			EncryptedPassword: password,
		},
	}
	if err := cli.Create(ctx, u); err != nil {
		return nil, err
	}

	workspaceRoleBinding := &iam.WorkspaceRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				iam.UserReferenceLabel: opt.StudentID,
				tenant.WorkspaceLabel:  MainWorkspace,
			},
			Name: opt.StudentID + "-" + MainWorkspaceViewer,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: ApiGroupIAM,
			Kind:     iam.ResourceKindWorkspaceRole,
			Name:     MainWorkspaceViewer,
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup: ApiGroupRBACAuthorization,
				Kind:     KindUser,
				Name:     opt.StudentID,
			},
		},
	}
	if err := cli.Create(ctx, workspaceRoleBinding); err != nil {
		return nil, err
	}

	return &User{
		Name:     u.Name,
		StringID: u.Name,
	}, nil
}

func (cli KSCli) CreateProject(ctx context.Context, opt *CreateProjectOpt) (*Project, error) {
	opt.ProjectName = "project-" + strings.ToLower(opt.ProjectName)

	var err error
	defer func() {
		if err != nil {
			// 强行兜底
			_ = cli.deleteProject(ctx, opt.ProjectName)
		}
	}()

	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        opt.ProjectName,
			Annotations: AdminCreatorAnnotation,
			Labels: map[string]string{
				tenant.WorkspaceLabel: MainWorkspace,
			},
		},
	}
	if err = cli.Create(ctx, ns); err != nil {
		return nil, err
	}

	// 设置Namespace中的资源限额
	quota := &v1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:        opt.ProjectName + "-ns-resource-limit",
			Namespace:   opt.ProjectName,
			Annotations: AdminCreatorAnnotation,
		},
		Spec: v1.ResourceQuotaSpec{
			Hard: map[v1.ResourceName]resource.Quantity{
				v1.ResourceName("limits.cpu"):    resource.MustParse("2"),
				v1.ResourceName("limits.memory"): resource.MustParse("4096Mi"),
			},
		},
	}
	if err = cli.Create(ctx, quota); err != nil {
		return nil, err
	}

	// 设置容器默认的CPU和内存占用
	limitRange := &v1.LimitRange{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   opt.ProjectName,
			Name:        opt.ProjectName + "-default-container-resource-limit",
			Annotations: AdminCreatorAnnotation,
		},
		Spec: v1.LimitRangeSpec{
			Limits: []v1.LimitRangeItem{
				{
					Default: map[v1.ResourceName]resource.Quantity{
						v1.ResourceCPU:    resource.MustParse("0.25"),
						v1.ResourceMemory: resource.MustParse("500Mi"),
					},
					Type: v1.LimitTypeContainer,
				},
			},
		},
	}
	if err = cli.Create(ctx, limitRange); err != nil {
		return nil, err
	}

	// 创建 harbor registry

	return &Project{Name: opt.ProjectName}, nil
}

func (cli KSCli) deleteProject(ctx context.Context, projectName string) error {
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: projectName},
	}
	return cli.Delete(ctx, ns)
}

func (cli KSCli) DeleteProject(ctx context.Context, project *Project) error {
	return cli.deleteProject(ctx, project.Name)
}

func (cli KSCli) AddOwner(_ context.Context, user *User, project *Project) error {
	rc, err := kube.NewRClient(cli.username, cli.password, cli.url)
	if err != nil {
		return err
	}
	return rc.AddProjectMember(project.Name, user.Name, "operator")
}

func (cli KSCli) RemoveMember(ctx context.Context, u *User, p *Project) error {
	panic("implement me")
}
