package db

import (
	"errors"
	"time"

	"github.com/bugitt/git-module"
	gouuid "github.com/satori/go.uuid"
	"gopkg.in/yaml.v3"
	log "unknwon.dev/clog/v2"
	"xorm.io/xorm"
)

type PipeStage int

const (
	NotStart       PipeStage = iota + 1 // 1
	LoadRepoStart                       // 2
	LoadRepoEnd                         // 3
	PreBuildStart                       // 4
	PreBuildEnd                         // 5
	BuildStart                          // 6
	BuildEnd                            // 7
	PostBuildStart                      // 8
	PostBuildEnd                        // 9
	PushStart                           // 10
	PushEnd                             // 11
	DeployStart                         // 12
	DeployEnd                           // 13
)

type RunStatus int

const (
	BeforeStart RunStatus = iota + 1
	Running
	Finished
)

type PipeType string

const (
	PUSH   PipeType = "push"
	PR     PipeType = "pr"
	MANUAL PipeType = "manual"
)

type Pipeline struct {
	ID           int64
	PusherID     int64 // 推送者，表示谁触发了该Pipeline的创建
	RepoID       int64
	ProjectID    int64 // 项目ID
	UUID         string
	RefName      string
	Commit       string
	ImageTag     string
	PipeType     PipeType
	ConfigString string `xorm:"text"`
	CIPath       string
	IsSuccessful bool
	Log          string `xorm:"text"`
	ErrMsg       string `xorm:"text"`
	Status       RunStatus
	Stage        PipeStage
	TaskNum      int
	BeginUnix    int64 // 开始时间戳
	EndUnix      int64 // 结束时间戳
	BaseModel    `xorm:"extends"`
}

type BasicTaskResult struct {
	ID           int64
	Name         string
	Describe     string `xorm:"text"`
	PipelineID   int64
	IsSuccessful bool
	Log          string `xorm:"text"`
	ErrMsg       string `xorm:"text"`
	Duration     int64
	BeginUnix    int64 // 开始时间戳
	EndUnix      int64 // 结束时间戳
}

type PreBuildResult struct {
	Number          int
	BasicTaskResult `xorm:"extends"`
	BaseModel       `xorm:"extends"`
}

type PostBuildResult struct {
	Number          int
	BasicTaskResult `xorm:"extends"`
	BaseModel       `xorm:"extends"`
}

type BuildResult struct {
	ImageTag        string `xorm:"text"`
	BasicTaskResult `xorm:"extends"`
	BaseModel       `xorm:"extends"`
}

type PushResult struct {
	ImageTag        string
	BasicTaskResult `xorm:"extends"`
	BaseModel       `xorm:"extends"`
}

type DeployResult struct {
	IP              string
	Ports           string `xorm:"TEXT"`
	Namespace       string
	DeploymentName  string
	ServiceName     string
	LogStart        int64 // 查询日志时，假定的开始时间
	BasicTaskResult `xorm:"extends"`
	BaseModel       `xorm:"extends"`
}

func SaveCIResult(result interface{}) error {
	_, err := x.Insert(result)
	return err
}

func (pipeline *Pipeline) Begin() error {
	pipeline.BeginUnix = time.Now().Unix()
	pipeline.Status = Running
	_, err := x.ID(pipeline.ID).Update(pipeline)
	return err
}

func (pipeline *Pipeline) Succeed() error {
	pipeline.IsSuccessful = true
	pipeline.EndUnix = time.Now().Unix()
	pipeline.Status = Finished
	row, err := x.ID(pipeline.ID).Update(pipeline)
	if err == nil && row != 1 {
		err = errors.New("set ptask success failed")
	}
	return err
}

func (pipeline *Pipeline) Fail(sourceErr error) error {
	pipeline.IsSuccessful = false
	pipeline.EndUnix = time.Now().Unix()
	pipeline.ErrMsg = sourceErr.Error()
	pipeline.Status = Finished
	row, err := x.ID(pipeline.ID).Update(pipeline)
	if err == nil && row != 1 {
		err = errors.New("update ptask failed")
	}
	return err
}

func (pipeline *Pipeline) UpdateStage(status PipeStage, taskNum int) error {
	pipeline.Stage = status
	pipeline.TaskNum = taskNum
	pipeline.Status = Running
	_, err := x.Where("id = ?", pipeline.ID).Update(pipeline)
	return err
}

func (result *BasicTaskResult) End(begin time.Time, err error, logs string) {
	result.BeginUnix = begin.Unix()
	result.Duration = time.Since(begin).Milliseconds()
	result.EndUnix = time.Now().Unix()
	if err != nil {
		result.ErrMsg = err.Error()
	} else {
		result.IsSuccessful = true
	}
	result.Log = logs
}

func PreparePipeline(commit *git.Commit, pipeType PipeType, repo *Repository, pusher *User, refName string, conf *CIConfig, confErr error) (*Pipeline, error) {
	pipeline := &Pipeline{
		RepoID:    repo.ID,
		PusherID:  pusher.ID,
		RefName:   refName,
		PipeType:  pipeType,
		Commit:    commit.ID.String(),
		ImageTag:  genImageTag(repo, commit.ID.String()),
		ProjectID: repo.OwnerID,
		Stage:     NotStart,
		Status:    BeforeStart,
		TaskNum:   -1,
	}

	if confErr != nil {
		pipeline.ErrMsg = confErr.Error()
		pipeline.IsSuccessful = false
		pipeline.Status = Finished
	} else {
		conf.Pretty()
		confS, _ := yaml.Marshal(conf)
		pipeline.ConfigString = string(confS)
	}

	id, err := createPipeline(x, pipeline)
	if err != nil {
		log.Error("%s", err.Error())
		return nil, err
	}
	pipeline.ID = id
	return pipeline, nil
}

func createPipeline(e Engine, p *Pipeline) (int64, error) {
	p.UUID = gouuid.NewV4().String()
	_, err := e.Insert(p)
	if err != nil {
		return -1, err
	}
	return p.ID, nil
}

func (pipeline *Pipeline) BeforeInsert() {
	pipeline.BaseModel.BeforeInsert()
}

func (pipeline *Pipeline) AfterSet(colName string, cell xorm.Cell) {
	pipeline.BaseModel.AfterSet(colName, cell)
}

func GetNotStartPipelines(repoID int64) ([]*Pipeline, error) {
	tasks := make([]*Pipeline, 0)
	query := x.Where("stage = ?", NotStart).And("status <= ?", BeforeStart)
	if repoID > 0 {
		query = query.And("repo_id = ?", repoID)
	}
	err := query.Find(&tasks)
	return tasks, err
}

func GetPipelinesByRepo(repoID int64) ([]*Pipeline, error) {
	ps := make([]*Pipeline, 0)
	err := x.Where("repo_id = ?", repoID).Find(&ps)
	return ps, err
}

func GetPipelinesByRepoList(repoIDs []int64) ([]*Pipeline, error) {
	ps := make([]*Pipeline, 0)
	err := x.In("repo_id", repoIDs).Find(&ps)
	return ps, err
}

func GetPipelinesByIDList(ids []int64) ([]*Pipeline, error) {
	ps := make([]*Pipeline, 0)
	err := x.In("id", ids).Find(&ps)
	return ps, err
}

func GetPipeline(repoID int64, commit string) (*Pipeline, error) {
	pipeline := &Pipeline{
		RepoID: repoID,
		Commit: commit,
	}
	has, err := x.Get(pipeline)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, nil
	}
	return pipeline, nil
}

// GetLatestPipeline 获取最新的Pipeline
func GetLatestPipeline(repoID int64) (*Pipeline, error) {
	pipeline := &Pipeline{
		RepoID: repoID,
	}
	has, err := x.OrderBy("updated_unix desc").Get(pipeline)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, nil
	}
	return pipeline, nil
}

func PrettyStage(stage PipeStage) string {
	des := ""
	switch stage {
	case DeployEnd:
		des = "已部署完成"
	case DeployStart:
		des = "正在部署中……"
	case PushEnd:
		des = "推送镜像已完成"
	case PushStart:
		des = "正在推送镜像中……"
	case PostBuildEnd:
		des = "已完成测试"
	case PostBuildStart:
		des = "正在测试中……"
	case BuildEnd:
		des = "镜像构建完成"
	case BuildStart:
		des = "镜像构建中……"
	case PreBuildEnd:
		des = "静态代码检查已完成"
	case PreBuildStart:
		des = "静态代码检查中……"
	case LoadRepoEnd:
		des = "仓库文件准备完成……"
	case LoadRepoStart:
		des = "准备仓库文件中……"
	case NotStart:
		des = "等待开始……"
	}
	return des
}
