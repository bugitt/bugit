package db

import "strings"

type CloudUser struct {
	ID           string
	Name         string
	NickName     string
	Email        string
	Role         int
	DepartmentID string
	IsAccept     bool
	AcceptTime   string
}

func (s CloudUser) Table() string {
	return "user"
}

func ExistCloudUser(studentID string) (bool, error) {
	return cloudX.Table("user").Where("id = ? ", strings.ToLower(studentID)).Exist()
}
