package db

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/conf"
	"git.scs.buaa.edu.cn/iobs/bugit/internal/tool"
	"github.com/artdarek/go-unzip"
	"github.com/bugitt/git-module"
	gouuid "github.com/satori/go.uuid"
	"github.com/unknwon/com"
	"gopkg.in/yaml.v3"
	log "unknwon.dev/clog/v2"
	"xorm.io/xorm"
)

type PipeStage int

const (
	NotStart      PipeStage = iota + 1 // 1
	LoadRepoStart                      // 2
	LoadRepoEnd                        // 3
	ValidStart                         // 4
	ValidEnd                           // 5
	BuildStart                         // 6
	BuildEnd                           // 7
	TestStart                          // 8
	TestEnd                            // 9
	PushStart                          // 10
	PushEnd                            // 11
	DeployStart                        // 12
	DeployEnd                          // 13
)

type RunStatus int

const (
	BeforeStart RunStatus = iota + 1
	Running
	Finished
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
	ConfigString string `xorm:"text"`
	CIPath       string
	IsSuccessful bool
	Log          string `xorm:"text"`
	ErrMsg       string `xorm:"text"`
	Status       RunStatus
	Stage        PipeStage
	BeginUnix    int64 // 开始时间戳
	EndUnix      int64 // 结束时间戳
	BaseModel    `xorm:"extends"`
}

type BasicTask struct {
	ID           int64
	PipeTaskID   int64
	Number       int
	Status       RunStatus
	IsSuccessful bool
	ErrType      CIErrType
	SrcErrMsg    string `xorm:"text"`
	CusErrMsg    string `xorm:"text"`
	BeginUnix    int64  // 开始时间戳
	EndUnix      int64  // 结束时间戳
}

func GetNotStartPipelines(repoID int64) ([]*Pipeline, error) {
	tasks := make([]*Pipeline, 0)
	query := x.Where("stage = ?", NotStart)
	if repoID > 0 {
		query = query.And("repo_id = ?", repoID)
	}
	err := query.Find(&tasks)
	return tasks, err
}

func (pipeline *Pipeline) prepareValidstaionTask(index int) (*ValidationTask, error) {
	task := &ValidationTask{}
	task.PipeTaskID = pipeline.ID
	task.Number = index
	task.Status = BeforeStart
	_, err := x.Insert(task)
	return task, err
}

func (pipeline *Pipeline) prepareBuildTask(context *CIContext, index int) (*BuildTask, error) {
	task := &BuildTask{}
	task.PipeTaskID = pipeline.ID
	task.Number = index
	task.Status = BeforeStart
	task.ImageTag = context.imageTag
	_, err := x.Insert(task)
	return task, err
}

func (pipeline *Pipeline) preparePushTask(context *CIContext) (*PushTask, error) {
	task := &PushTask{}
	task.PipeTaskID = pipeline.ID
	task.Status = BeforeStart
	_, err := x.Insert(task)
	return task, err
}

func (pipeline *Pipeline) prepareDeployTask(ctx *CIContext) (*DeployTask, error) {
	task := &DeployTask{}
	task.PipeTaskID = pipeline.ID
	task.Status = BeforeStart
	task.NameSpace = fmt.Sprintf("%d", ctx.owner.ID)
	task.DeploymentName = ctx.repo.DeployName() + "-deployment"
	task.ServiceName = ctx.repo.DeployName() + "-service"
	_, err := x.Insert(task)
	return task, err
}

func (pipeline *Pipeline) Validation(ctx *CIContext) error {
	_ = pipeline.updateStatus(ValidStart)
	configs := pipeline.Pipeline.Config.Validate
	for i := range configs {
		task, err := pipeline.prepareValidstaionTask(i + 1)
		if err != nil {
			return err
		}
		_ = task.start()
		if err = task.Run(ctx); err != nil {
			_ = task.failed()
			return err
		}
		_ = task.success()
	}
	return pipeline.updateStatus(ValidEnd)
}

func (pipeline *Pipeline) Build(ctx *CIContext) error {
	_ = pipeline.updateStatus(BuildStart)
	task, err := pipeline.prepareBuildTask(ctx, 1)
	if err != nil {
		return err
	}
	_ = task.start()
	if err = task.Run(ctx); err != nil {
		_ = task.failed()
		return err
	}
	_ = task.success()

	return pipeline.updateStatus(BuildEnd)
}

func (pipeline *Pipeline) Push(ctx *CIContext) error {
	_ = pipeline.updateStatus(PushStart)
	task, err := pipeline.preparePushTask(ctx)
	if err != nil {
		return err
	}
	_ = task.start()
	if err = task.Run(ctx); err != nil {
		_ = task.failed()
		return err
	}
	_ = task.success()
	return pipeline.updateStatus(PushEnd)
}

func (pipeline *Pipeline) Deploy(ctx *CIContext) error {
	_ = pipeline.updateStatus(DeployStart)
	task, err := pipeline.prepareDeployTask(ctx)
	if err != nil {
		return err
	}
	_ = task.start()
	if err = task.Run(ctx); err != nil {
		_ = task.failed()
		return err
	}
	_ = task.success()
	return pipeline.updateStatus(DeployEnd)
}

func preparePipeTask(pipeline *Pipeline, pusher *User) error {
	pipeTask := &PipeTask{
		PipelineID: pipeline.ID,
		RepoID:     pipeline.RepoID,
		SenderID:   pusher.ID,
		ImageTag:   pipeline.ImageTag,
		Stage:      NotStart,
		Status:     BeforeStart,
		ProjectID:  pipeline.ProjectID,
	}
	return createPipeTask(x, pipeTask)
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

func GetPipeTasksByProject(projectID int64) ([]*PipeTask, error) {
	ps := make([]*PipeTask, 0)
	err := x.Where("project_id = ?", projectID).Find(&ps)
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

// GetLatestPipeTask 查找最新的pipeTask
func GetLatestPipeTask(pipelineID int64) (*PipeTask, error) {
	pipeTask := &PipeTask{
		PipelineID: pipelineID,
	}
	has, err := x.OrderBy("created_unix desc").Get(pipeTask)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, nil
	}
	return pipeTask, nil
}

func IsPipelineRunning(repoID int64, commit string) (bool, error) {
	pipeline, err := GetPipeline(repoID, commit)
	if err != nil {
		return false, err
	}
	if pipeline == nil {
		return false, nil
	}
	ptask, err := GetLatestPipeTask(pipeline.ID)
	if err != nil {
		return false, err
	}
	if ptask == nil {
		return false, nil
	}
	return ptask.Status == BeforeStart || ptask.Status == Running, nil
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
	case TestEnd:
		des = "已完成测试"
	case TestStart:
		des = "正在测试中……"
	case BuildEnd:
		des = "镜像构建完成"
	case BuildStart:
		des = "镜像构建中……"
	case ValidEnd:
		des = "静态代码检查已完成"
	case ValidStart:
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

func createPipeTask(e Engine, p *PipeTask) error {
	p.UUID = gouuid.NewV4().String()
	_, err := e.Insert(p)
	return err
}

func (ptask *PipeTask) LoadRepo(context *CIContext) error {
	err := ptask.updateStatus(LoadRepoStart)
	if err != nil {
		return nil
	}
	if err := ptask.Pipeline.loadRepo(); err != nil {
		return err
	}
	return ptask.updateStatus(LoadRepoEnd)
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
	row, err := x.ID(pipeline.ID).Cols("is_succeed", "status", "end_unix").Update(pipeline)
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
	row, err := x.ID(pipeline.ID).Cols("is_succeed", "status", "end_unix", "err_msg", "err_type").Update(pipeline)
	if err == nil && row != 1 {
		err = errors.New("update ptask failed")
	}
	return err
}

func (pipeline *Pipeline) updateStatus(status PipeStage) error {
	pipeline.Stage = status
	_, err := x.Where("id = ?", pipeline.ID).Update(pipeline)
	return err
}

func (pipeline *Pipeline) GetDeployTask() (dtask *DeployTask, err error) {
	dtask = &DeployTask{
		BasicTask: BasicTask{
			PipeTaskID: pipeline.ID,
		},
	}
	has, err := x.OrderBy("created_unix desc").Get(dtask)
	if err != nil {
		return
	}
	if !has {
		return nil, nil
	}
	return
}

func PreparePipeline(commit *git.Commit, configS []byte, repo *Repository, pusher *User, refName string) (*Pipeline, error) {
	imageTag := fmt.Sprintf("%s/%s/%s:%s",
		conf.Docker.Registry,
		repo.MustOwner().LowerName,
		repo.LowerName,
		commit.ID.String()[:5])
	pipeline := &Pipeline{
		RepoID:       repo.ID,
		PusherID:     pusher.ID,
		RefName:      refName,
		Commit:       commit.ID.String(),
		ImageTag:     imageTag,
		ConfigString: string(configS),
		ProjectID:    repo.OwnerID,
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
	// 先检查一下是不是已经创建过相同的pipeline配置了
	oldPipe := &Pipeline{
		RepoID: p.RepoID,
		Commit: p.Commit,
	}
	has, err := x.Get(oldPipe)
	if err != nil {
		return -1, err
	}
	if has {
		// 如果有旧的，就更新一下
		p.ID = oldPipe.ID
		_, err = e.ID(p.ID).Update(p)
	} else {
		_, err = e.Insert(p)
	}
	if err != nil {
		return -1, err
	}
	return p.ID, nil
}

func (pipeline *Pipeline) loadAttributes() error {
	if pipeline.repoDB == nil {
		repo := new(Repository)
		has, err := x.ID(pipeline.RepoID).Get(repo)
		if err != nil {
			return err
		}
		if !has {
			return errors.New("repo not found")
		}
		pipeline.repoDB = repo
	}

	if pipeline.gitRepo == nil {
		gitRepo, err := git.Open(pipeline.repoDB.RepoPath())
		if err != nil {
			return err
		}
		pipeline.gitRepo = gitRepo
	}

	if pipeline.gitCommit == nil {
		gitCommit, err := pipeline.gitRepo.CatFileCommit(pipeline.Commit)
		if err != nil {
			return err
		}
		pipeline.gitCommit = gitCommit
	}

	return nil
}

func (pipeline *Pipeline) loadRepo() error {
	// 如果已经存在了，那么就不用再load一次了
	if com.IsDir(pipeline.CIPath) {
		return nil
	}
	hash := tool.ShortSHA1(pipeline.Commit)
	archivePath := filepath.Join(pipeline.gitRepo.Path(), "archives", "zip")
	if !com.IsDir(archivePath) {
		if err := os.MkdirAll(archivePath, os.ModePerm); err != nil {
			return err
		}
	}
	archivePath = path.Join(archivePath, hash+".zip")
	if !com.IsFile(archivePath) {
		if err := pipeline.gitCommit.CreateArchive(git.ArchiveZip, archivePath); err != nil {
			return err
		}
	}

	repoPath := filepath.Join(conf.Devops.Tmpdir, pipeline.repoDB.MustOwner().Name, pipeline.repoDB.Name, hash)
	if !com.IsDir(repoPath) {
		uz := unzip.New(archivePath, repoPath)
		if err := uz.Extract(); err != nil {
			return err
		}
	}
	pipeline.CIPath = repoPath
	// 更新数据库
	_, err := x.ID(pipeline.ID).Update(pipeline)
	return err
}

func (pipeline *Pipeline) BeforeInsert() {
	pipeline.BaseModel.BeforeInsert()

	// 保证 configString 不为空
	if len(pipeline.ConfigString) <= 0 {
		configS, err := yaml.Marshal(pipeline.Config)
		if err != nil {
			log.Error("%s, marchel config string error for %#v", err.Error(), pipeline)
			return
		}
		pipeline.ConfigString = string(configS)
	}
}

func (pipeline *Pipeline) AfterSet(colName string, cell xorm.Cell) {
	pipeline.BaseModel.AfterSet(colName, cell)
	switch colName {
	case "config_string":
		pipeline.Config, _ = ParseCIConfig([]byte(pipeline.ConfigString))
	}
}
