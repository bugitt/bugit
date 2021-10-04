package kube

import (
	"strings"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/db/errors"
	"github.com/go-resty/resty/v2"
	iamv1alpha2 "kubesphere.io/api/iam/v1alpha2"
)

type Token struct {
	// AccessToken is the token that authorizes and authenticates
	// the requests.
	AccessToken string `json:"access_token"`

	// TokenType is the type of token.
	// The Type method returns either this or "Bearer", the default.
	TokenType string `json:"token_type,omitempty"`

	// RefreshToken is a token that's used by the application
	// (as opposed to the user) to refresh the access token
	// if it expires.
	RefreshToken string `json:"refresh_token,omitempty"`

	// ID Token value associated with the authenticated session.
	IDToken string `json:"id_token,omitempty"`

	// ExpiresIn is the optional expiration second of the access token.
	ExpiresIn int `json:"expires_in,omitempty"`
}

type Member struct {
	Username string `json:"username"`
	RoleRef  string `json:"roleRef"`
}

type RestClient struct {
	*resty.Client
}

func NewRClient(username, password, url string) (*RestClient, error) {
	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		url = "http://" + url
	}
	cli := resty.New()
	cli.SetHostURL(url)
	token := &Token{}
	_, err := cli.R().
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetFormData(map[string]string{
			"grant_type": "password",
			"username":   username,
			"password":   password,
		}).
		SetResult(token).
		Post("/oauth/token")
	if err != nil {
		return nil, err
	}
	cli.SetAuthToken(token.AccessToken)
	return &RestClient{cli}, nil
}

func (cli RestClient) AddProjectMember(ns, username, role string) error {
	memberList := []Member{{username, role}}
	_, err := cli.R().
		SetHeader("Content-Type", "application/json").
		SetBody(&memberList).
		SetPathParam("namespace", ns).
		Post("/kapis/iam.kubesphere.io/v1alpha2/namespaces/{namespace}/members")
	return err
}

func (cli RestClient) GetProjectMember(ns, username string) (*iamv1alpha2.User, error) {
	u := &iamv1alpha2.User{}
	resp, err := cli.R().
		SetHeader("Content-Type", "application/json").
		SetPathParam("namespace", ns).
		SetPathParam("username", username).
		SetResult(u).
		Get("/kapis/iam.kubesphere.io/v1alpha2/namespaces/{namespace}/members/{username}")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != 200 {
		return nil, errors.New(resp.String())
	}
	return u, err
}

func (cli RestClient) DeleteProjectMember(ns, username string) error {
	_, err := cli.R().
		SetHeader("Content-Type", "application/json").
		SetBody(&struct{}{}).
		SetPathParam("namespace", ns).
		SetPathParam("username", username).
		Delete("/kapis/iam.kubesphere.io/v1alpha2/namespaces/{namespace}/members/{username}")
	return err
}
