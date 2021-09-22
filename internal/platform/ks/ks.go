package ks

import (
	"strings"

	"github.com/go-resty/resty/v2"
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
