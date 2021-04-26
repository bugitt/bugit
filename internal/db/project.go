package db

import (
	"fmt"
)

type ProjectStatus int

type ProjectList []*Project

type Project struct {
	ID         int64
	Name       string        `xorm:"INDEX NOT NULL" gorm:"NOT NULL" json:"name"`
	SenderID   int64         `xorm:"UNIQUE(s) INDEX NOT NULL" gorm:"UNIQUE_INDEX:s;NOT NULL" json:"sender_id"`
	Sender     *User         `xorm:"-" gorm:"-"`
	ExpID      int64         `xorm:"UNIQUE(s) INDEX NOT NULL" gorm:"UNIQUE_INDEX:s;NOT NULL" json:"exp_id"`
	ExpString  string        `json:"exp_name"`
	CourseID   int64         `xorm:"INDEX NOT NULL" gorm:"NOT NULL" json:"course_id"`
	CourseName string        `json:"course_name"`
	Status     ProjectStatus `json:"status"`
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

func GetProject(project *Project) error {
	has, err := x.Get(project)
	if err != nil {
		return err
	} else if !has {
		return ErrProjectNotExist{project}
	}
	return project.LoadAttributes()
}

func GetProjectsByCourseIDList(courseIDList []int64) ([]*Project, error) {
	data := make([]*Project, 0)
	err := x.In("course_id", courseIDList).Find(&data)
	return data, err
}

func GetUserAllProjects(user *User) (ProjectList, error) {
	return GetUserProjects(&UserProjectOptions{
		SenderID: user.ID,
		Page:     1,
		PageSize: user.NumProjects,
	})
}

func GetUsersAllProjectsByUserList(userIDList []int64) (ProjectList, error) {
	projects := make([]*Project, 0)
	err := x.In("sender_id", userIDList).Find(&projects)
	return projects, err
}

func (p *Project) LoadAttributes() error {
	return p.loadAttributes(x)
}

func (p *Project) GetMembers() (members []*User, err error) {
	err = p.LoadAttributes()
	if err != nil {
		return
	}
	members = make([]*User, 0)
	if !p.Sender.IsOrganization() {
		members = append(members, p.Sender)
		return
	}
	org := p.Sender
	err = org.GetMembers(1 << 30)
	if err != nil {
		return
	}
	members = org.Members
	return
}

// TODO: 防止多次查询
func (p *Project) loadAttributes(e Engine) (err error) {
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

func GetAllProjectsWithCoAndAttr(user *User) ([]*Project, error) {
	projects, err := GetUserAllProjects(user)
	if err != nil {
		return nil, err
	}
	err = user.GetOrganizations(true)
	if err != nil {
		return nil, err
	}

	// 对org列表进行索引
	orgMap := make(map[int64]*User)
	orgIDList := make([]int64, 0, len(user.Orgs))
	for _, org := range user.Orgs {
		orgMap[org.ID] = org
		orgIDList = append(orgIDList, org.ID)
	}
	ps, err := GetUsersAllProjectsByUserList(orgIDList)
	if err != nil {
		return nil, err
	}
	for i := range ps {
		ps[i].Sender = orgMap[ps[i].SenderID]
	}
	projects = append(projects, ps...)

	for i := range projects {
		err = projects[i].LoadAttributes()
		if err != nil {
			return nil, err
		}
	}
	return projects, nil
}
