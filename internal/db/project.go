package db

import (
	"fmt"
	"io/ioutil"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/conf"
	"git.scs.buaa.edu.cn/iobs/bugit/internal/httplib"
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
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

func GetProjectByID(id int64) (*Project, error) {
	project := &Project{}
	has, err := x.ID(id).Get(project)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, ErrProjectNotExist{project}
	}
	err = project.LoadAttributes()
	if err != nil {
		return nil, err
	}
	return project, nil
}

func GetProjectsByCourseIDList(courseIDList []int64) ([]*Project, error) {
	data := make(ProjectList, 0)
	err := x.In("course_id", courseIDList).Find(&data)
	if err != nil {
		return nil, err
	}
	err = data.LoadAttributes()
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

func (p *Project) GetRepos() ([]*Repository, error) {
	repos := make([]*Repository, 0)
	err := x.Where("project_id = ?", p.ID).Find(&repos)
	return repos, err
}

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

func (p *Project) HomeLink() string {
	return fmt.Sprintf("%s/project/%d", conf.Server.Subpath, p.ID)
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

func GetCourseIDListByToken(token string) ([]int64, error) {
	resp, err := httplib.Get("http://vlab.beihangsoft.cn/api/user/getCoursesByUser").
		Header("Authorization", token).
		Response()
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	type CloudResp struct {
		Code int     `json:"code"`
		Msg  string  `json:"msg"`
		Data []int64 `json:"data"`
	}
	cloudResp := &CloudResp{}
	err = jsoniter.Unmarshal(body, cloudResp)
	if err != nil {
		return nil, err
	}
	if cloudResp.Code != 1001 {
		return nil, errors.New("error verifying user information")
	}
	return cloudResp.Data, nil
}

func (p *Project) GetDeployList(repos ...*Repository) ([]*DeployDes, error) {
	var err error
	if len(repos) <= 0 {
		repos, err = p.GetRepos()
		if err != nil {
			return nil, err
		}
	}
	deps := make([]*DeployDes, 0, len(repos))
	for _, repo := range repos {
		des, err := GetDeploy(repo)
		if err != nil && !IsErrPipeNotFound(err) {
			return nil, err
		}
		if !IsErrPipeNotFound(err) {
			deps = append(deps, des)
		}
	}
	return deps, err
}
