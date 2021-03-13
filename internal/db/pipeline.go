package db

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/artdarek/go-unzip"
	"github.com/bugitt/git-module"
	gouuid "github.com/satori/go.uuid"
	"github.com/unknwon/com"
	"gogs.io/gogs/internal/conf"
	"gogs.io/gogs/internal/tool"
	"gopkg.in/yaml.v3"
	log "unknwon.dev/clog/v2"
	"xorm.io/xorm"
)

type PipeStage int

const (
	NotStart PipeStage = iota + 1
	LoadRepoStart
	LoadRepoEnd
	ValidStart
	ValidEnd
	BuildStart
	BuildEnd
	TestStart
	TestEnd
	PushStart
	PushEnd
	DeployStart
	DeployEnd
)

type Pipeline struct {
	ID           int64
	PusherID     int64 // 推送者，表示谁触发了该Pipeline的创建
	RepoID       int64
	repoDB       *Repository     `xorm:"-" json:"-"`
	gitRepo      *git.Repository `xorm:"-" json:"-"`
	UUID         string
	RefName      string
	Commit       string
	gitCommit    *git.Commit `xorm:"-" json:"-"`
	ConfigString string
	Config       *CIConfig `xorm:"-" json:"-"`
	CIPath       string
	BaseModel    `xorm:"extends"`
}

type PipeTask struct {
	ID         int64
	RepoID     int64
	UUID       string
	PipelineID int64
	Pipeline   *Pipeline `xorm:"-" json:"-"`
	SenderID   int64     // 表示是谁触发了这次pipeline的执行
	SenderTime int64     // 触发执行时的时间戳
	Stage      PipeStage
	IsSucceed  bool
	BeginUnix  int64 // 开始时间戳
	EndUnix    int64 // 结束时间戳
	BaseModel  `xorm:"extends"`
}

func preparePipeTask(pipeline *Pipeline, pusher *User) error {
	pipeTask := &PipeTask{
		PipelineID: pipeline.ID,
		RepoID:     pipeline.RepoID,
		SenderID:   pusher.ID,
		Stage:      NotStart,
	}
	return createPipeTask(x, pipeTask)
}

func createPipeTask(e Engine, p *PipeTask) error {
	p.UUID = gouuid.NewV4().String()
	_, err := e.Insert(p)
	return err
}

func (ptask *PipeTask) LoadRepo() error {
	err := ptask.updateStatus(LoadRepoStart)
	if err != nil {
		return nil
	}
	if err := ptask.Pipeline.loadRepo(); err != nil {
		return err
	}
	return ptask.updateStatus(LoadRepoEnd)
}

func (ptask *PipeTask) CI() (err error) {
	// prepare attributes
	if err := ptask.loadAttributes(); err != nil {
		return err
	}

	defer func() {
		// 保证打上结束的时间戳
		_ = ptask.endTime()
		if err == nil {
			err = ptask.success()
		}
	}()

	if err = ptask.beginTime(); err != nil {
		return
	}

	switch ptask.Pipeline.Config.Version {
	case "0.0.1":
		err = ptask.CI001()
	}

	return err
}

func (ptask *PipeTask) beginTime() error {
	ptask.BeginUnix = time.Now().Unix()
	_, err := x.ID(ptask.ID).Update(ptask)
	return err
}

func (ptask *PipeTask) endTime() error {
	ptask.EndUnix = time.Now().Unix()
	_, err := x.ID(ptask.ID).Update(ptask)
	return err
}

func (ptask *PipeTask) success() error {
	ptask.IsSucceed = true
	row, err := x.ID(ptask.ID).Cols("is_succeed").Update(ptask)
	if err == nil && row != 1 {
		err = errors.New("set ptask success failed")
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

func preparePipeline(commit *git.Commit, configS []byte, repo *Repository, pusher *User, refName string) (*Pipeline, error) {
	pipeline := &Pipeline{
		RepoID:       repo.ID,
		PusherID:     pusher.ID,
		RefName:      refName,
		Commit:       commit.ID.String(),
		ConfigString: string(configS),
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
		return oldPipe.ID, nil
	}

	p.UUID = gouuid.NewV4().String()
	_, err = e.Insert(p)
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
