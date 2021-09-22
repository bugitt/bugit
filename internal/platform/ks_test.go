package platform

import (
	"context"
	"reflect"
	"testing"

	"github.com/loheagn/ksclient/client"
)

func TestKSCli_CreateUser(t *testing.T) {
	type fields struct {
		Client client.Client
	}
	type args struct {
		ctx context.Context
		opt *CreateUserOpt
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *User
		wantErr bool
	}{
		{
			name:   "just test create user",
			fields: fields{Client: NewKSCli("10.251.0.40:31889", "admin", "qAs.wChKwF5iKf#4")},
			args: args{
				ctx: context.Background(),
				opt: &CreateUserOpt{
					StudentID: "15131057",
					UserName:  "wurilege",
					Email:     "15131057@buaa.edu.cn",
					RealName:  "wurilege",
					Password:  "newpass",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := KSCli{
				Client: tt.fields.Client,
			}
			got, err := cli.CreateUser(tt.args.ctx, tt.args.opt)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			t.Log(got)
		})
	}
}

func TestKSCli_CreateProject(t *testing.T) {
	type fields struct {
		Client client.Client
	}
	type args struct {
		ctx context.Context
		opt *CreateProjectOpt
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		want       *Project
		wantErr    bool
		wantDelErr bool
	}{
		{
			name:   "simple create simple project",
			fields: fields{Client: NewKSCli("10.251.0.40:31889", "admin", "qAs.wChKwF5iKf#4")},
			args: args{
				ctx: context.Background(),
				opt: &CreateProjectOpt{ProjectName: "15131059"},
			},
			want: &Project{Name: "15131059"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := KSCli{
				Client: tt.fields.Client,
			}
			got, err := cli.CreateProject(tt.args.ctx, tt.args.opt)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateProject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateProject() got = %v, want %v", got, tt.want)
			}

			// 测试删除
			delErr := cli.DeleteProject(tt.args.ctx, got)
			if (delErr != nil) != tt.wantDelErr {
				t.Errorf("DeleteProject() error = %v, wantErr %v", delErr, tt.wantDelErr)
				return
			}
		})
	}
}
