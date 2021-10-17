package kube

import (
	"context"
	"testing"
)

func TestPodDeploy(t *testing.T) {
	type args struct {
		ctx context.Context
		opt *PodDeployOpt
	}
	cli, err := NewClient(context.Background(), &CreateClientOpt{
		ConfigPath: "",
		Namespace:  "default",
	})
	if err != nil {
		panic(err.Error())
	}
	timeS := "2021-8-11-99"
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "simple-test",
			args: args{
				ctx: context.Background(),
				opt: &PodDeployOpt{
					Labels: map[string]string{
						"simple": "test",
						"time":   timeS,
					},
					ExtraLabels: map[string]string{
						"inner": "pod56",
					},
					ReplicaNum: 5,
					Stateful:   false,
					Duration:   0,
					Spec: PodSpec{
						Name:     "simple-test",
						ImageTag: "harbor.scs.buaa.edu.cn/library/nginx:1.17",
						Envs:     nil,
						Ports: []Port{
							{
								Name: "main",
								Port: 80,
							},
						},
						WorkDir: "",
						Cmd:     Cmd{},
						Quota: Quota{
							CPU:    "1",
							Memory: "2G",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := cli.PodDeploy(tt.args.ctx, tt.args.opt); (err != nil) != tt.wantErr {
				t.Errorf("PodDeploy() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
