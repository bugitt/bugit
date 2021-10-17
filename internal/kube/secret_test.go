package kube

import (
	"context"
	"testing"
)

func TestClient_EnsureDockerRegistry(t *testing.T) {
	cli, err := NewClient(context.Background(), &CreateClientOpt{
		ConfigPath: "",
		Namespace:  "default",
	})
	if err != nil {
		panic(err.Error())
	}
	type args struct {
		ctx context.Context
		p   *PrivateDockerRegistrySecret
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "bugit-test",
			args: args{
				ctx: context.Background(),
				p: &PrivateDockerRegistrySecret{
					Name:     "bugit-test",
					Username: "admin",
					Password: "HarRuanjianbor&&12345",
					Host:     "harbor.scs.buaa.edu.cn",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := cli.CreateOrUpdateDockerRegistrySecret(tt.args.ctx, tt.args.p); (err != nil) != tt.wantErr {
				t.Errorf("CreateOrUpdateDockerRegistrySecret() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
