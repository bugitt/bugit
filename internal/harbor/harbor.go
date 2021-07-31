package harbor

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

func getInt64Ptr(x int64) *int64 {
	return &x
}

func getClient() (*apiv2.RESTClient, error) {
	return apiv2.NewRESTClientForHost(conf.Harbor.Url, conf.Harbor.AdminName, conf.Harbor.AdminPasswd)
}

func getUP(ctx context.Context, client *apiv2.RESTClient, projectID int64, uName string) (*legacymodel.User, *modelv2.Project, error) {
	u, err := client.GetUser(ctx, uName)
	if err != nil {
		return nil, nil, err
	}
	p, err := client.GetProject(ctx, strconv.FormatInt(projectID, 10))
	return u, p, err
}

// CreateUser post a user to harbor
func CreateUser(ctx context.Context, username, email, realname string) (int64, error) {
	client, err := getClient()
	if err != nil {
		return 0, err
	}

	// check whether user already exists
	_, err = client.GetUser(ctx, username)
	if err != nil {
		if _, ok := err.(*user.ErrUserNotFound); !ok {
			return 0, err
		}
	} else {
		return 0, nil
	}

	// create
	u, err := client.NewUser(ctx, username, email, realname, conf.Harbor.DefaultPasswd, "User created by BuGit")
	return u.UserID, err
}

func CreateProject(ctx context.Context, name, username string) (int64, error) {
	client, err := getClient()
	if err != nil {
		return 0, err
	}

	// check whether project already exists
	// if yes, delete it
	p, err := client.GetProject(ctx, name)
	if err != nil {
		if _, ok := err.(*project.ErrProjectNotFound); !ok {
			return 0, err
		}
	} else {
		if err = client.DeleteProject(ctx, p); err != nil {
			return 0, err
		}
	}

	p, err = client.NewProject(ctx, name, getInt64Ptr(-1))
	if err != nil {
		return 0, err
	}

	// 将BuGit中的项目的创建这作为管理员，添加到Harbor项目中
	u, err := client.GetUser(ctx, username)
	if err != nil {
		return 0, err
	}

	// The role id 1 for projectAdmin, 2 for developer, 3 for guest, 4 for maintainer
	return int64(p.ProjectID), client.AddProjectMember(ctx, p, u, 1)
}

// AddProjectMember 默认添加的每个member都有管理员权限
func AddProjectMember(ctx context.Context, projectID int64, uName string) error {
	client, err := getClient()
	if err != nil {
		return err
	}
	u, p, err := getUP(ctx, client, projectID, uName)
	if err != nil {
		return err
	}
	return client.AddProjectMember(ctx, p, u, 1)
}

func DeleteProjectMember(ctx context.Context, projectID int64, uName string) error {
	client, err := getClient()
	if err != nil {
		return err
	}
	u, p, err := getUP(ctx, client, projectID, uName)
	if err != nil {
		return err
	}
	return client.DeleteProjectMember(ctx, p, u)
}

func DeleteProject(ctx context.Context, projectID string) error {
	client, err := getClient()
	if err != nil {
		return err
	}
	p, err := client.GetProject(ctx, projectID)
	if err != nil {
		return err
	}
	return client.DeleteProject(ctx, p)
}
