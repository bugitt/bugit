package harbor

import (
	"context"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/conf"
	"github.com/mittwald/goharbor-client/v4/apiv2"
	"github.com/mittwald/goharbor-client/v4/apiv2/user"
)

func getClient() (*apiv2.RESTClient, error) {
	return apiv2.NewRESTClientForHost(conf.Harbor.Url, conf.Harbor.AdminName, conf.Harbor.AdminPasswd)
}

// CreateUser post a user to harbor
func CreateUser(ctx context.Context, username, email, realname string) error {
	client, err := getClient()
	if err != nil {
		return err
	}

	// check whether user already exists
	_, err = client.GetUser(ctx, username)
	if err != nil {
		if _, ok := err.(*user.ErrUserNotFound); !ok {
			return err
		}
	} else {
		return nil
	}

	// create
	_, err = client.NewUser(ctx, username, email, realname, conf.Harbor.DefaultPasswd, "User created by BuGit")
	return err
}
