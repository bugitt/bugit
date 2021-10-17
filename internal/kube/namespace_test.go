package kube

import (
	"context"
	"testing"
)

func TestClient_EnsureNS(t *testing.T) {
	// 创建 Client
	namespace := "test-ns"
	cli, err := NewClient(context.Background(), &CreateClientOpt{
		ConfigPath: "",
		Namespace:  namespace,
	})
	if err != nil {
		panic(err.Error())
	}
	type args struct {
		quota Quota
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test-namespace",
			args: args{quota: Quota{
				CPU:    "3",
				Memory: "6g",
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				_ = cli.DeleteNS()
			}()
			if err := cli.EnsureNS(tt.args.quota); (err != nil) != tt.wantErr {
				t.Errorf("EnsureNS() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
