package db

import (
	"fmt"
	"time"

	"xorm.io/xorm"
)

type ProjectStatus int

type ProjectList []*Project

type Project struct {
	ID          int64
	Name        string `xorm:"INDEX NOT NULL" gorm:"NOT NULL"`
	SenderID    int64  `xorm:"UNIQUE(s) INDEX NOT NULL" gorm:"UNIQUE_INDEX:s;NOT NULL"`
	Sender      *User  `xorm:"-" gorm:"-" json:"-"`
	ExpID       int64  `xorm:"UNIQUE(s) INDEX NOT NULL" gorm:"UNIQUE_INDEX:s;NOT NULL"`
	ExpString   string
	CourseID    int64 `xorm:"INDEX NOT NULL" gorm:"NOT NULL"`
	CourseName  string
	Status      ProjectStatus
	CreatedUnix int64
	Created     time.Time `xorm:"-" gorm:"-" json:"-"`
	UpdatedUnix int64
	Updated     time.Time `xorm:"-" gorm:"-" json:"-"`
}

type UserProjectOptions struct {
	SenderID int64
	Page     int
	PageSize int
}

// GetUserProjects returns a list of projects of given user.
func GetUserProjects(opts *UserProjectOptions) (ProjectList, error) {
	sess := x.Where("sender_id=?", opts.SenderID).Desc("updated_unix")

	if opts.Page <= 0 {
		opts.Page = 1
	}
	sess.Limit(opts.PageSize, (opts.Page-1)*opts.PageSize)

	projects := make([]*Project, 0, opts.PageSize)
	return projects, sess.Find(&projects)
}

func (ps ProjectList) LoadAttributes() error {
	return ps.loadAttributes(x)
}

func (p *Project) BeforeInsert() {
	p.CreatedUnix = time.Now().Unix()
	p.UpdatedUnix = p.CreatedUnix
}

func (p *Project) BeforeUpdate() {
	p.UpdatedUnix = time.Now().Unix()
}

func (p *Project) AfterSet(colName string, _ xorm.Cell) {
	switch colName {
	case "created_unix":
		p.Created = time.Unix(p.CreatedUnix, 0).Local()
	case "updated_unix":
		p.Updated = time.Unix(p.UpdatedUnix, 0)
	}
}

func (ps ProjectList) loadAttributes(e Engine) error {
	// Get Users
	userSet := make(map[int64]*User)
	for i := range ps {
		userSet[ps[i].SenderID] = nil
	}
	userIDs := make([]int64, 0, len(userSet))
	for userID := range userSet {
		userIDs = append(userIDs, userID)
	}
	users := make([]*User, 0, len(userIDs))
	if err := e.Where("id > 0").In("id", userIDs).Find(&users); err != nil {
		return fmt.Errorf("find users: %v", err)
	}
	for i := range users {
		userSet[users[i].ID] = users[i]
	}
	for i := range ps {
		ps[i].Sender = userSet[ps[i].SenderID]
	}
	return nil
}
