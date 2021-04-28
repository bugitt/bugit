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
	courseIDList, err := getCourseIDListByToken(c.Token())
	if err != nil {
		log.Error(err.Error())
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	projects, err := db.GetProjectsByCourseIDList(courseIDList)
	if err != nil {
		log.Error(err.Error())
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSONSuccess(projects)
}

func CreateProject(c *context.APIContext, form CreateOption) {
	senderID := c.User.ID
	if form.OrgName != "" {
		user, err := db.GetUserByName(form.OrgName)
		if err != nil && !db.IsErrUserNotExist(err) {
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

	// 先找到这个project
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

	authOK, err := authForAccessProject(c, project)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	if !authOK {
		c.JSON(http.StatusForbidden, "no permission to read the content of this project")
		return
	}

	// 获取该project中的成员
	members, err := project.GetMembers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSONSuccess(members)
}

func ListRepos(c *context.APIContext) {
	projectID := c.ParamsInt64("projectID")
	if projectID <= 0 {
		c.JSON(http.StatusBadRequest, "param error: can not parse projectID from this url")
		return
	}

	// 先找到这个project
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

	authOK, err := authForAccessProject(c, project)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	if !authOK {
		c.JSON(http.StatusForbidden, "no permission to read the content of this project")
		return
	}

	// 获取该project中的成员
	repos, err := project.GetRepos()
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	for i := range repos {
		err = repos[i].LoadBranches()
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}
	}
	c.JSONSuccess(repos)
}

func CreateDeploy(c *context.APIContext, opt db.DeployOption) {
	projectID := c.ParamsInt64("projectID")
	if projectID <= 0 {
		c.JSON(http.StatusBadRequest, "param error: can not parse projectID from this url")
		return
	}

	// 先找到这个project
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

	authOK, err := authForAccessProject(c, project)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	if !authOK {
		c.JSON(http.StatusForbidden, "no permission to read the content of this project")
		return
	}

	opt.Repo, err = db.GetRepositoryByID(opt.RepoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	authOK, err = authForAccessRepo(c, opt.Repo, project)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	if !authOK {
		c.JSON(http.StatusForbidden, "no permission to deploy this repo, at least write permission is required")
		return
	}

	// 好了，终于可以触发部署了
	opt.Pusher = c.User
	err = db.CreateDeploy(&opt)
	if err != nil {
		if db.IsErrNoNeedDeploy(err) || db.IsErrNoValidCIConfig(err) {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}

	c.Status(http.StatusOK)
}

func getCourseIDListByToken(token string) ([]int64, error) {
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

func authForAccessProject(c *context.APIContext, project *db.Project) (bool, error) {
	// 如果该project是个人的项目
	if c.UserID() == project.SenderID {
		return true, nil
	}

	// 如果该project是组织的项目
	if org := project.Sender; org.IsOrganization() && org.IsOrgMember(c.UserID()) {
		return true, nil
	}

	// 如果是管理员
	return checkProjectCloudAdmin(c, project)
}

func authForAccessRepo(c *context.APIContext, repo *db.Repository, project *db.Project) (bool, error) {
	// 普通用户的鉴权
	accessMode := db.Perms.AccessMode(c.UserID(), repo.ID,
		db.AccessModeOptions{
			OwnerID: repo.OwnerID,
			Private: repo.IsPrivate,
		},
	)
	if accessMode >= db.AccessModeWrite {
		return true, nil
	}

	// 如果是管理员的话
	ok, err := checkProjectCloudAdmin(c, project)
	if err != nil {
		return false, err
	}
	return ok && repo.ProjectID == project.ID, nil
}

// checkProjectCloudAdmin 该操作非常耗时，请谨慎调用
func checkProjectCloudAdmin(c *context.APIContext, project *db.Project) (ok bool, err error) {
	if c.IsProjectAdmin != context.ProjectAdminNotSure {
		return c.IsProjectAdmin == context.ProjectAdminTrue, nil
	}
	courseIDList, err := getCourseIDListByToken(c.Token())
	if err != nil {
		return false, err
	}
	for _, courseID := range courseIDList {
		if project.CourseID == courseID {
			ok = true
			break
		}
	}
	if ok {
		c.IsProjectAdmin = context.ProjectAdminTrue
	} else {
		c.IsProjectAdmin = context.ProjectAdminFalse
	}
	return
}
