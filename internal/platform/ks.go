package platform

import (
	"context"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/conf"
	"github.com/loheagn/ksclient/client"
	"github.com/loheagn/ksclient/client/generic"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	iam "kubesphere.io/api/iam/v1alpha2"
	tenant "kubesphere.io/api/tenant/v1alpha1"
)

type KSCli struct {
	client.Client
}

var (
	_ = &client.URLOptions{
		Group:   "iam.kubesphere.io",
		Version: "v1alpha2",
	}
)

const (
	MainWorkspace             = "main-workspace"
	MainWorkspaceViewer       = "main-workspace-viewer"
	ApiGroupIAM               = "iam.kubesphere.io"
	ApiGroupRBACAuthorization = "rbac.authorization.k8s.io"
	KindUser                  = "User"
)

func NewKSCli(url, adminName, adminPassword string) *KSCli {
	config := &rest.Config{
		Host:     url,
		Username: adminName,
		Password: adminPassword,
	}
	if err := iam.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		panic(err)
	}
	cli := generic.NewForConfigOrDie(config, client.Options{Scheme: scheme.Scheme})
	return &KSCli{cli}
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
	panic("implement me")
}

func (cli KSCli) DeleteProject(ctx context.Context, project *Project) error {
	panic("implement me")
}

func (cli KSCli) AddOwner(ctx context.Context, user *User, project *Project) error {
	panic("implement me")
}

func (cli KSCli) RemoveMember(ctx context.Context, u *User, p *Project) error {
	panic("implement me")
}
