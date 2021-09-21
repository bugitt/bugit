package platform

import (
	"context"
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
					StudentID: "15131052",
					UserName:  "wurilege",
					Email:     "15131052@buaa.edu.cn",
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
