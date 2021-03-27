package db

import "testing"

func Test_ensureNS(t *testing.T) {
	type args struct {
		pid int64
		uid int64
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{args: args{pid: 7, uid: 10}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ensureNS(tt.args.pid, tt.args.uid); (err != nil) != tt.wantErr {
				t.Errorf("ensureNS() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
