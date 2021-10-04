package platform

import (
	"context"
	"reflect"
	"testing"
)

func getTestCli() *KSCli {
	return NewKSCli(
		"172.16.8.1:32698",
		"admin",
		"qAs.wChKwF5iKf#4",
		"harbor.scs.buaa.edu.cn",
		"admin",
		"HarRuanjianbor&&12345",
	)
}

func TestKSCli_CreateUser(t *testing.T) {
	type args struct {
		ctx context.Context
		opt *CreateUserOpt
	}
	tests := []struct {
		name    string
		args    args
		want    *User
		wantErr bool
	}{
		{
			name: "just test create user",
			args: args{
				ctx: context.Background(),
				opt: &CreateUserOpt{
					StudentID: "15131059",
					Email:     "15131057@buaa.edu.cn",
					RealName:  "wurilege",
					Password:  "newpass",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := getTestCli()
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
	type args struct {
		ctx         context.Context
		projectName string
	}
	tests := []struct {
		name       string
		args       args
		want       *Project
		wantErr    bool
		wantDelErr bool
	}{
		{
			name: "simple create and delete simple project",
			args: args{
				ctx:         context.Background(),
				projectName: "15131059",
			},
			want: &Project{Name: "project-15131059"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := getTestCli()
			got, err := cli.CreateProject(tt.args.ctx, tt.args.projectName)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateProject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateProject() got = %v, want %v", got, tt.want)
			}

			//测试删除
			delErr := cli.DeleteProject(tt.args.ctx, got)
			if (delErr != nil) != tt.wantDelErr {
				t.Errorf("DeleteProject() error = %v, wantErr %v", delErr, tt.wantDelErr)
				return
			}
		})
	}
}

func TestKSCli_AddOwner(t *testing.T) {
	type args struct {
		in0     context.Context
		user    *User
		project *Project
	}
	tests := []struct {
		name       string
		args       args
		wantErr    bool
		wantDelErr bool
	}{
		{
			name: "simple add member",
			args: args{
				in0:     nil,
				user:    &User{Name: "15131057"},
				project: &Project{Name: "project-15131059"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := getTestCli()
			if err := cli.AddOwner(tt.args.in0, tt.args.user, tt.args.project); (err != nil) != tt.wantErr {
				t.Errorf("AddOwner() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err := cli.RemoveMember(tt.args.in0, tt.args.user, tt.args.project); (err != nil) != tt.wantDelErr {
				t.Errorf("RemoveMember() error = %v, wantErr %v", err, tt.wantDelErr)
			}
		})
	}
}
