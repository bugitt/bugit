package db

import "strings"

type CloudUser struct {
	ID           string `xorm:"pk"`
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

func GetAllCloudUserList() ([]*CloudUser, error) {
	users := make([]*CloudUser, 0)
	err := cloudX.Table("user").Where("1=1").Find(&users)
	return users, err
}

func UpdateCloudUserEmail(u *CloudUser) error {
	_, err := cloudX.ID(u.ID).Cols("email").Update(u)
	return err
}
