package kube

import (
	"encoding/base64"
	"fmt"

	json "github.com/json-iterator/go"
	v1 "k8s.io/api/core/v1"
)

type HarborOpt struct {
	Username string
	Password string
	Host     string
}

func GenDockerRegistrySecret(opt *HarborOpt) (*v1.Secret, error) {
	configJson := map[string]map[string]DockerRegistryAuthConfig{
		"auths": {
			opt.Host: DockerRegistryAuthConfig{
				Username: opt.Username,
				Password: opt.Password,
				Auth:     base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", opt.Username, opt.Password))),
			},
		},
	}
	data, _ := json.Marshal(configJson)
	return &v1.Secret{
		Immutable:  boolPtr(false),
		StringData: map[string]string{".dockerconfigjson": string(data)},
		Type:       v1.SecretTypeDockerConfigJson,
	}, nil
}

func boolPtr(b bool) *bool {
	return &b
}
