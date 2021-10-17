package kube

import (
	"context"
	"encoding/json"
	"fmt"

	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PrivateDockerRegistrySecret struct {
	Name     string
	Username string
	Password string
	Host     string
}

type SecretOpt struct {
	Name       string
	Immutable  bool
	Data       map[string][]byte
	StringData map[string]string
	Type       v1.SecretType
}

func (opt SecretOpt) convertToSecret() *v1.Secret {
	return &v1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: opt.Name,
		},
		Immutable:  &opt.Immutable,
		Data:       opt.Data,
		StringData: opt.StringData,
		Type:       opt.Type,
	}
}

type DockerRegistryAuthConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Auth     string `json:"auth"`
}

func (p PrivateDockerRegistrySecret) convertToSecret() *SecretOpt {
	configJson := map[string]map[string]DockerRegistryAuthConfig{
		"auths": {
			p.Host: DockerRegistryAuthConfig{
				Username: p.Username,
				Password: p.Password,
				Auth:     base64EncodeString([]byte(fmt.Sprintf("%s:%s", p.Username, p.Password))),
			},
		},
	}
	data, _ := json.Marshal(configJson)
	return &SecretOpt{
		Name:       p.Name,
		Immutable:  false,
		StringData: map[string]string{".dockerconfigjson": string(data)},
		Type:       v1.SecretTypeDockerConfigJson,
	}
}

func (cli *Client) CreateOrUpdateDockerRegistrySecret(ctx context.Context, p *PrivateDockerRegistrySecret) (err error) {
	s, err := cli.CoreV1().Secrets(cli.namespace).Get(ctx, p.Name, metav1.GetOptions{})
	if err != nil && kerrors.IsNotFound(err) {
		return cli.CrateSecret(ctx, p.convertToSecret())
	}
	if err != nil {
		return err
	}
	if s.Immutable != nil && *s.Immutable {
		return nil
	}
	return cli.UpdateSecret(ctx, p.convertToSecret())
}

func (cli *Client) UpdateSecret(ctx context.Context, opt *SecretOpt) (err error) {
	_, err = cli.CoreV1().Secrets(cli.namespace).Update(ctx, opt.convertToSecret(), metav1.UpdateOptions{})
	return
}

func (cli *Client) CrateSecret(ctx context.Context, opt *SecretOpt) (err error) {
	_, err = cli.CoreV1().Secrets(cli.namespace).Create(ctx, opt.convertToSecret(), metav1.CreateOptions{})
	return err
}
