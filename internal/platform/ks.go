package platform

import (
	"context"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/conf"
	"github.com/loheagn/ksclient/client"
	"github.com/loheagn/ksclient/client/generic"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	iam "kubesphere.io/api/iam/v1alpha2"
)

type KSCli struct {
	client.Client
}

var (
	iamUrlOpt = &client.URLOptions{
		Group:   "iam.kubesphere.io",
		Version: "v1alpha2",
	}
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
	err := cli.Create(ctx, u)
	if err != nil {
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
