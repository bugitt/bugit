package db

type ProjectStatus int

type Project struct {
	ID         int64
	Name       string `xorm:"INDEX NOT NULL" gorm:"NOT NULL"`
	SenderID   int64  `xorm:"UNIQUE(s) INDEX NOT NULL" gorm:"UNIQUE_INDEX:s;NOT NULL"`
	Sender     *User  `xorm:"-" gorm:"-" json:"-"`
	ExpID      int64  `xorm:"UNIQUE(s) INDEX NOT NULL" gorm:"UNIQUE_INDEX:s;NOT NULL"`
	ExpString  string
	CourseID   int64 `xorm:"INDEX NOT NULL" gorm:"NOT NULL"`
	CourseName string
	Status     ProjectStatus
}
