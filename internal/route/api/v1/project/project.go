package project

import (
	"errors"
	"io/ioutil"
	"net/http"

	jsoniter "github.com/json-iterator/go"
	log "unknwon.dev/clog/v2"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/context"
	"git.scs.buaa.edu.cn/iobs/bugit/internal/db"
	"git.scs.buaa.edu.cn/iobs/bugit/internal/httplib"
)

// CreateOption 创建project时可以提供的参数
type CreateOption struct {
	Name       string `json:"projectName" form:"projectName" binding:"Required"`
	OrgName    string `json:"organizationName" form:"organizationName"`
	ExpName    string `json:"expName" form:"expName" binding:"Required"`
	ExpID      int64  `json:"expId" form:"expId" binding:"Required"`
	CourseName string `json:"courseName" form:"courseName" binding:"Required"`
	CourseID   int64  `json:"courseId" form:"courseId" binding:"Required"`
}

func GetAllProjects(c *context.APIContext) {
	projects, err := db.GetAllProjectsWithCoAndAttr(c.User)
	if err != nil {
		log.Error(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}
	c.JSONSuccess(projects)
}

func GetProjectsByCourse(c *context.APIContext) {
	courseListS := c.Query("courseIds")
	var (
		projects []*db.Project
		err      error
	)
	if courseListS == "-1" {
		projects, err = db.GetUserAllProjects(c.User)
		if err != nil {
			log.Error(err.Error())
			c.Status(http.StatusInternalServerError)
			return
		}
		c.JSONSuccess(projects)
	}
	courseIDList := make([]int64, 0)
	err = jsoniter.Unmarshal([]byte(courseListS), &courseIDList)
	if err != nil {
		log.Error(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}
	projects, err = db.GetProjectsByUserAndCourse(c.UserID(), courseIDList)
	if err != nil {
		log.Error(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}
	c.JSONSuccess(projects)
}

func CreateProject(c *context.APIContext, form CreateOption) {
	senderID := c.User.ID
	if form.OrgName != "" {
		user, err := db.GetUserByName(form.OrgName)
		if err != nil {
			log.Error(err.Error())
			c.JSON(http.StatusInternalServerError, "error: get org")
			return
		}
		if user == nil {
			// 创建新的org
			if !c.User.CanCreateOrganization() {
				c.JSON(http.StatusBadRequest, "error: this user can not create org")
				return
			}
			org := &db.User{
				Name:     form.OrgName,
				IsActive: true,
				Type:     db.UserOrganization,
			}
			if err := db.CreateOrganization(org, c.User); err != nil {
				c.JSON(http.StatusBadRequest, "error: create org")
				return
			}
			log.Trace("Organization created: %s", org.Name)

			senderID = org.ID
		} else {
			senderID = user.ID
		}
	}
	project := &db.Project{
		Name:       form.Name,
		SenderID:   senderID,
		ExpID:      form.ExpID,
		ExpString:  form.ExpName,
		CourseID:   form.CourseID,
		CourseName: form.CourseName,
	}
	if err := db.CreateProject(project); err != nil {
		log.Error(err.Error())
		c.JSON(http.StatusBadRequest, "please check for duplicate: (senderID, expID)")
		return
	}

	c.JSON(http.StatusCreated, project)
}

func ListMembers(c *context.APIContext) {
	projectID := c.ParamsInt64("projectID")
	if projectID <= 0 {
		c.JSON(http.StatusBadRequest, "param error: can not parse projectID from this url")
		return
	}

	// 先找到这个project可能属于的org
	project := &db.Project{
		ID: projectID,
	}
	err := db.GetProject(project)
	if err != nil {
		if db.IsProjectNotExist(err) {
			c.JSON(http.StatusNotFound, "can not found this project")
			return
		}
	}

	// 如果该project不属于一个org，那么其相关的用户只有sender一个，直接判断权限，然后返回就好
	if !project.Sender.IsOrganization() {
		// 确认一下，该用户是否对该项目有权限
		if project.SenderID != c.UserID() {
			c.JSON(http.StatusForbidden, "no permission to read the content of this project")
			return
		}
		c.JSONSuccess([]*db.User{project.Sender})
		return
	}
	org := project.Sender

	// 检查该用户是否是该org的成员
	if !org.IsOrgMember(c.UserID()) {
		c.JSON(http.StatusForbidden, "no permission to read the content of this project")
		return
	}

	err = org.GetMembers(1 << 30)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}

	c.JSONSuccess(org.Members)
}

func getCourseIDListByToken(token string) ([]int64, error) {
	resp, err := httplib.Get("http://vlab.beihangsoft.cn/api/user/getCoursesByUser").
		Header("Authorization", "4efd14aa-74ad-44d8-a633-57a66708bb13").
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
