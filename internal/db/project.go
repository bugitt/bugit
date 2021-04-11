package db

import (
	"fmt"
)

type ProjectStatus int

type ProjectList []*Project

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
	BaseModel  `xorm:"extends"`
}

func CreateProject(project *Project) (err error) {
	_, err = x.Insert(project)
	return
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

func GetProjectByID(id int64) (*Project, error) {
	project := &Project{
		ID: id,
	}
	has, err := x.Where("id = ?", id).Get(project)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, ErrProjectNotExist{project}
	}
	return project, project.LoadAttributes()
}

func GetUserAllProjects(user *User) (ProjectList, error) {
	return GetUserProjects(&UserProjectOptions{
		SenderID: user.ID,
		Page:     1,
		PageSize: user.NumProjects,
	})
}

func (p Project) LoadAttributes() error {
	return p.loadAttributes(x)
}

func (p Project) loadAttributes(e Engine) (err error) {
	// Get User
	if p.Sender == nil {
		p.Sender, err = getUserByID(e, p.SenderID)
		if err != nil {
			return fmt.Errorf("getUserByID [%d]: %v", p.SenderID, err)
		}
	}

	return nil
}

func (ps ProjectList) LoadAttributes() error {
	return ps.loadAttributes(x)
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
