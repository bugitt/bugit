package db

import (
	"github.com/bugitt/git-module"
	gouuid "github.com/satori/go.uuid"
	"gopkg.in/yaml.v3"
	log "unknwon.dev/clog/v2"
	"xorm.io/xorm"
)

type PipeStage int

const (
	NotStart PipeStage = iota - 1
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
	UUID         string
	RefName      string
	Commit       string
	ConfigString string
	Config       *CIConfig `xorm:"-" json:"-"`
	BaseModel    `xorm:"extends"`
}

type PipeTask struct {
	ID         int64
	RepoID     int64
	RepoDB     *Repository `xorm:"-" json:"-"`
	UUID       string
	PipelineID int64
	Pipeline   *Pipeline `xorm:"-" json:"-"`
	SenderID   int64     // 表示是谁触发了这次pipeline的执行
	SenderTime int64     // 触发执行时的时间戳
	Stage      PipeStage
	IsSucceed  bool
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
	return e.Insert(p)
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
	case "config":
		p.Config, _ = ParseCIConfig([]byte(p.ConfigString))
	}
}
