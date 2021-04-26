package project

import (
	"reflect"
	"testing"
)

func Test_getCourseIDListByToken(t *testing.T) {
	type args struct {
		token string
	}
	tests := []struct {
		name    string
		args    args
		want    []int64
		wantErr bool
	}{
		{args: args{"4efd14aa-74ad-44d8-a633-57a66708bb13"}, want: []int64{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getCourseIDListByToken(tt.args.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("getCourseIDListByToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getCourseIDListByToken() = %v, want %v", got, tt.want)
			}
		})
	}
}
