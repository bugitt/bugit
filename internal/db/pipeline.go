package db

import (
	"context"
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

type CIContext struct {
	context.Context
	path     string
	imageTag string
	owner    *User
	repo     *Repository
	commit   string
	refName  string
	config   *CIConfig
}

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
	ProjectID    int64
	repoDB       *Repository     `xorm:"-" json:"-"`
	gitRepo      *git.Repository `xorm:"-" json:"-"`
	UUID         string
	RefName      string
	Commit       string
	ImageTag     string
	gitCommit    *git.Commit `xorm:"-" json:"-"`
	ConfigString string      `xorm:"text"`
	Config       *CIConfig   `xorm:"-" json:"-"`
	CIPath       string
	BaseModel    `xorm:"extends"`
}

type PipeTask struct {
	ID         int64
	RepoID     int64
	UUID       string
	PipelineID int64
	ProjectID  int64
	Pipeline   *Pipeline `xorm:"-" json:"-"`
	SenderID   int64     // 表示是谁触发了这次pipeline的执行
	SenderTime int64     // 触发执行时的时间戳
	Stage      PipeStage
	IsSucceed  bool
	ErrType    CIErrType
	ErrMsg     string `xorm:"text"`
	ImageTag   string
	Status     RunStatus
	BeginUnix  int64 // 开始时间戳
	EndUnix    int64 // 结束时间戳
	BaseModel  `xorm:"extends"`
}

type BasicTask struct {
	ID         int64
	PipeTaskID int64
	Number     int
	Status     RunStatus
	IsSucceed  bool
	ErrType    CIErrType
	SrcErrMsg  string `xorm:"text"`
	CusErrMsg  string `xorm:"text"`
	BeginUnix  int64  // 开始时间戳
	EndUnix    int64  // 结束时间戳
}

func (ptask *PipeTask) prepareValidstaionTask(index int) (*ValidationTask, error) {
	task := &ValidationTask{}
	task.PipeTaskID = ptask.ID
	task.Number = index
	task.Status = BeforeStart
	_, err := x.Insert(task)
	return task, err
}

func (ptask *PipeTask) prepareBuildTask(context *CIContext, index int) (*BuildTask, error) {
	task := &BuildTask{}
	task.PipeTaskID = ptask.ID
	task.Number = index
	task.Status = BeforeStart
	task.ImageTag = context.imageTag
	_, err := x.Insert(task)
	return task, err
}

func (ptask *PipeTask) preparePushTask(context *CIContext) (*PushTask, error) {
	task := &PushTask{}
	task.PipeTaskID = ptask.ID
	task.Status = BeforeStart
	_, err := x.Insert(task)
	return task, err
}

func (ptask *PipeTask) prepareDeployTask(ctx *CIContext) (*DeployTask, error) {
	task := &DeployTask{}
	task.PipeTaskID = ptask.ID
	task.Status = BeforeStart
	task.NameSpace = fmt.Sprintf("%d", ctx.owner.ID)
	task.DeploymentName = ctx.repo.DeployName() + "-deployment"
	task.ServiceName = ctx.repo.DeployName() + "-service"
	_, err := x.Insert(task)
	return task, err
}

func (ptask *PipeTask) Validation(ctx *CIContext) error {
	_ = ptask.updateStatus(ValidStart)
	configs := ptask.Pipeline.Config.Validate
	for i := range configs {
		task, err := ptask.prepareValidstaionTask(i + 1)
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
	return ptask.updateStatus(ValidEnd)
}

func (ptask *PipeTask) Build(ctx *CIContext) error {
	_ = ptask.updateStatus(BuildStart)
	task, err := ptask.prepareBuildTask(ctx, 1)
	if err != nil {
		return err
	}
	_ = task.start()
	if err = task.Run(ctx); err != nil {
		_ = task.failed()
		return err
	}
	_ = task.success()

	return ptask.updateStatus(BuildEnd)
}

func (ptask *PipeTask) Push(ctx *CIContext) error {
	_ = ptask.updateStatus(PushStart)
	task, err := ptask.preparePushTask(ctx)
	if err != nil {
		return err
	}
	_ = task.start()
	if err = task.Run(ctx); err != nil {
		_ = task.failed()
		return err
	}
	_ = task.success()
	return ptask.updateStatus(PushEnd)
}

func (ptask *PipeTask) Deploy(ctx *CIContext) error {
	_ = ptask.updateStatus(DeployStart)
	task, err := ptask.prepareDeployTask(ctx)
	if err != nil {
		return err
	}
	_ = task.start()
	if err = task.Run(ctx); err != nil {
		_ = task.failed()
		return err
	}
	_ = task.success()
	return ptask.updateStatus(DeployEnd)
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

// CI CI过程中生成的error应该被自己消费掉
func (ptask *PipeTask) Run() {
	// 一个task最多只允许跑一小时
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()

	var err error

	defer func() {
		if err != nil {
			err = ptask.fail(err)
		}
		if err != nil {
			log.Error("update ptask error: err")
		}
	}()

	// prepare attributes
	if err = ptask.loadAttributes(); err != nil {
		return
	}

	if err = ptask.begin(); err != nil {
		return
	}

	// work
	done := make(chan error)
	go func() {
		done <- func() (err error) {
			defer func() {
				if panicErr := recover(); panicErr != nil {
					err = fmt.Errorf("panic occurred: %#v", panicErr)
				}
			}()
			switch ptask.Pipeline.Config.Version {
			case "0.0.1":
				err = ptask.CI001(ctx)
			}
			return err
		}()
	}()
	select {
	case err = <-done:
	case <-ctx.Done():
		err = &CIError{
			Type: TimeoutErrType,
			err:  ctx.Err(),
		}
	}

	// 保证打上结束的时间戳
	if err == nil {
		log.Info("pipe CI success: %d", ptask.ID)
		err = ptask.success()
	} else {
		log.Error("pipe CI error: %s", err.Error())
	}
}

func (ptask *PipeTask) begin() error {
	ptask.BeginUnix = time.Now().Unix()
	ptask.Status = Running
	_, err := x.ID(ptask.ID).Update(ptask)
	return err
}

func (ptask *PipeTask) success() error {
	ptask.IsSucceed = true
	ptask.EndUnix = time.Now().Unix()
	ptask.Status = Finished
	row, err := x.ID(ptask.ID).Cols("is_succeed", "status", "end_unix").Update(ptask)
	if err == nil && row != 1 {
		err = errors.New("set ptask success failed")
	}
	return err
}

func (ptask *PipeTask) fail(sourceErr error) error {
	ptask.IsSucceed = false
	ptask.EndUnix = time.Now().Unix()
	ptask.ErrMsg = sourceErr.Error()
	if ciErr, ok := sourceErr.(*CIError); ok {
		ptask.ErrType = ciErr.Type
	}
	ptask.Status = Finished
	row, err := x.ID(ptask.ID).Cols("is_succeed", "status", "end_unix", "err_msg", "err_type").Update(ptask)
	if err == nil && row != 1 {
		err = errors.New("update ptask failed")
	}
	return err
}

func (ptask *PipeTask) updateStatus(status PipeStage) error {
	ptask.Stage = status
	_, err := x.Where("id = ?", ptask.ID).Update(ptask)
	return err
}

func (ptask *PipeTask) loadAttributes() error {
	if ptask.Pipeline == nil {
		pipeline := new(Pipeline)
		has, err := x.ID(ptask.PipelineID).Get(pipeline)
		if err != nil {
			return err
		}
		if !has {
			return errors.New("pipeline not found")
		}
		if err := pipeline.loadAttributes(); err != nil {
			return err
		}
		ptask.Pipeline = pipeline
	}

	return nil
}

func (ptask *PipeTask) GetDeployTask() (dtask *DeployTask, err error) {
	dtask = &DeployTask{
		BasicTask: BasicTask{
			PipeTaskID: ptask.ID,
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

func preparePipeline(commit *git.Commit, configS []byte, repo *Repository, pusher *User, refName string) (*Pipeline, error) {
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

func (p *Pipeline) loadAttributes() error {
	if p.repoDB == nil {
		repo := new(Repository)
		has, err := x.ID(p.RepoID).Get(repo)
		if err != nil {
			return err
		}
		if !has {
			return errors.New("repo not found")
		}
		p.repoDB = repo
	}

	if p.gitRepo == nil {
		gitRepo, err := git.Open(p.repoDB.RepoPath())
		if err != nil {
			return err
		}
		p.gitRepo = gitRepo
	}

	if p.gitCommit == nil {
		gitCommit, err := p.gitRepo.CatFileCommit(p.Commit)
		if err != nil {
			return err
		}
		p.gitCommit = gitCommit
	}

	return nil
}

func (p *Pipeline) loadRepo() error {
	// 如果已经存在了，那么就不用再load一次了
	if com.IsDir(p.CIPath) {
		return nil
	}
	hash := tool.ShortSHA1(p.Commit)
	archivePath := filepath.Join(p.gitRepo.Path(), "archives", "zip")
	if !com.IsDir(archivePath) {
		if err := os.MkdirAll(archivePath, os.ModePerm); err != nil {
			return err
		}
	}
	archivePath = path.Join(archivePath, hash+".zip")
	if !com.IsFile(archivePath) {
		if err := p.gitCommit.CreateArchive(git.ArchiveZip, archivePath); err != nil {
			return err
		}
	}

	repoPath := filepath.Join(conf.Devops.Tmpdir, p.repoDB.MustOwner().Name, p.repoDB.Name, hash)
	if !com.IsDir(repoPath) {
		uz := unzip.New(archivePath, repoPath)
		if err := uz.Extract(); err != nil {
			return err
		}
	}
	p.CIPath = repoPath
	// 更新数据库
	_, err := x.ID(p.ID).Update(p)
	return err
}

func (p *Pipeline) BeforeInsert() {
	p.BaseModel.BeforeInsert()

	// 保证 configString 不为空
	if len(p.ConfigString) <= 0 {
		configS, err := yaml.Marshal(p.Config)
		if err != nil {
			log.Error("%s, marchel config string error for %#v", err.Error(), p)
			return
		}
		p.ConfigString = string(configS)
	}
}

func (p *Pipeline) AfterSet(colName string, cell xorm.Cell) {
	p.BaseModel.AfterSet(colName, cell)
	switch colName {
	case "config_string":
		p.Config, _ = ParseCIConfig([]byte(p.ConfigString))
	}
}
