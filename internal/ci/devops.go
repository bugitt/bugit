package ci

import (
	"fmt"
	"strconv"
	"time"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/conf"
	"git.scs.buaa.edu.cn/iobs/bugit/internal/db"
	"git.scs.buaa.edu.cn/iobs/bugit/internal/sync"
	"github.com/bugitt/git-module"
	log "unknwon.dev/clog/v2"
)

var CIQueue = sync.NewUniqueQueue(1000)

func getCIConfigFromCommit(commit *git.Commit) (*db.CIConfig, []byte, error) {
	var fileContent []byte
	var err error
	for _, filename := range conf.Devops.Filename {
		if fileContent, err = commit.ReadFileSimple(filename); err != nil {
			continue
		} else {
			break
		}
	}
	if len(fileContent) <= 0 {
		return nil, nil, nil
	}
	ciConfig, err := db.ParseCIConfig(fileContent)
	if err != nil {
		return nil, nil, err
	}
	return ciConfig, fileContent, nil
}

func ci() {
	// repoID = 0 表示获取所有没有开始执行的pipeline
	tasks, err := db.GetNotStartPipelines(0)
	if err != nil {
		log.Error("Get pre pipeline: %v", err)
	}
	for _, ptask := range tasks {
		go ptask.Run()

	}
	for repoID := range CIQueue.Queue() {
		log.Trace("Begin Pipeline for [repo_id: %v]", repoID)
		CIQueue.Remove(repoID)

		repoIDInt, err := strconv.ParseInt(repoID, 10, 64)
		if err != nil {
			log.Error("invalid repoID read from CIQueue %s", repoID)
			continue
		}
		tasks, err := db.GetNotStartPipelines(repoIDInt)
		if err != nil {
			log.Error("Get repository [%s] pipelines: %v", repoID, err)
			continue
		}
		for _, ptask := range tasks {
			go ptask.Run()
		}
	}
}

func StartCI() {
	go ci()
}

type DeployOption struct {
	RepoID    int64           `json:"RepoID" form:"RepoID" binding:"Required"`
	Repo      *db.Repository  `json:"-"`
	gitRepo   *git.Repository `json:"-"`
	Branch    string          `json:"Branch" form:"Branch"`
	Commit    string          `json:"Commit" form:"Commit"`
	gitCommit *git.Commit     `json:"-"`
	Pusher    *db.User        `json:"-"`
}

func CreateDeploy(opt *DeployOption) (err error) {
	// 1. 完善参数
	if opt.Repo != nil {
		opt.RepoID = opt.Repo.ID
	} else {
		repo, err := db.GetRepositoryByID(opt.RepoID)
		if err != nil {
			return err
		}
		opt.Repo = repo
	}
	opt.gitRepo, err = git.Open(opt.Repo.RepoPath())
	if err != nil {
		return err
	}

	if len(opt.Branch) <= 0 {
		opt.Branch = opt.Repo.DefaultBranch
	}

	if len(opt.Commit) <= 0 {
		opt.gitCommit, err = opt.gitRepo.BranchCommit(opt.Branch)
		if err != nil {
			return
		}
		opt.Commit = opt.gitCommit.ID.String()
	} else {
		opt.gitCommit, err = opt.gitRepo.CatFileCommit(opt.Commit)
		if err != nil {
			return
		}
	}

	// 2. 检查对应的commit中是否有合法的 CIConfig 配置文件
	ciConfig, fileContent, err := getCIConfigFromCommit(opt.gitCommit)
	if err != nil {
		return err
	}
	if ciConfig == nil {
		return &db.ErrNoValidCIConfig{
			RepoName: opt.Repo.Name,
			Branch:   opt.Branch,
			Commit:   opt.Commit,
		}
	}

	// 3. 首先检查是不是真的需要进行一次 deploy
	//    当前的计划是，如果当前有同样commit的deploy正在执行，那么就忽略本次部署请求
	isRunning, err := db.IsPipelineRunning(opt.RepoID, opt.Commit)
	if err != nil {
		return err
	}
	if isRunning {
		return &db.ErrNoNeedDeploy{"the deployment of the current commit is in progress, please stay calm"}
	}

	// 4. 好了，终于确定了，可以触发新的部署了
	_, err = db.PreparePipeline(opt.gitCommit, fileContent, opt.Repo, opt.Pusher, opt.Branch)
	if err != nil {
		return err
	}

	go CIQueue.Add(opt.Repo.ID)
	return nil
}

// DeployDes 描述一个project中的一个仓库最新的部署情况
type DeployDes struct {
	// 总体描述
	Repo         *db.Repository `json:"-"`
	RepoID       int64
	RepoName     string
	Branch       string
	BranchURL    string
	Commit       string
	CommitURL    string
	PrettyCommit string
	Status       db.RunStatus
	Stage        db.PipeStage
	StageString  string
	IsSuccessful bool
	IsHealthy    bool
	ErrMsg       string
	BeginUnix    int64
	EndUnix      int64
	Pusher       *db.User
	// 流水线任务创建的时间
	CreatedUnix int64
	Created     time.Time `json:"-"`

	// 具体的部署情况
	ImageTag   string
	IP         string
	HasDeploy  bool
	Namespace  string
	Deployment string
	Service    string
	PodLabels  map[string]string
	DepLabels  map[string]string
	SvcLabels  map[string]string
	Ports      []db.Port
}

func DescribePipeTask(pipeline *db.Pipeline, repos ...*db.Repository) (re *DeployDes, err error) {
	var repo *db.Repository
	if len(repos) <= 0 {
		repo, err = db.GetRepositoryByID(pipeline.RepoID)
		if err != nil {
			return
		}
	} else {
		repo = repos[0]
	}

	pusher, err := db.GetUserByID(pipeline.PusherID)
	if err != nil {
		return nil, err
	}

	re = &DeployDes{
		Repo:         repo,
		RepoID:       repo.ID,
		RepoName:     repo.Name,
		Branch:       pipeline.RefName,
		BranchURL:    fmt.Sprintf("%s/src/%s", repo.Link(), pipeline.RefName),
		Commit:       pipeline.Commit,
		CommitURL:    fmt.Sprintf("%s/commit/%s", repo.Link(), pipeline.Commit),
		PrettyCommit: pipeline.Commit[:10],
		Status:       pipeline.Status,
		Stage:        pipeline.Stage,
		StageString:  db.PrettyStage(pipeline.Stage),
		IsSuccessful: pipeline.IsSucceed,
		ErrMsg:       pipeline.ErrMsg,
		BeginUnix:    pipeline.BeginUnix,
		EndUnix:      pipeline.EndUnix,
		Pusher:       pusher,
		CreatedUnix:  pipeline.CreatedUnix,
		Created:      time.Unix(pipeline.CreatedUnix, 0),

		ImageTag: pipeline.ImageTag,
	}

	dtask, err := pipeline.GetDeployTask()
	if err != nil {
		return
	}
	if dtask == nil {
		re.HasDeploy = false
		re.IsHealthy = re.IsSuccessful
		return
	}

	// 下面处理部署部分的内容
	re.Namespace = dtask.NameSpace
	re.IP = dtask.IP
	re.Deployment = dtask.DeploymentName
	re.Service = dtask.ServiceName
	re.Ports = dtask.GetPorts()
	re.PodLabels = db.GetPodLabels(repo, pipeline.RefName, pipeline.Commit)
	re.DepLabels = db.GetSvcLabels(repo)
	re.SvcLabels = re.DepLabels

	// 检查已经部署的各个resource是否 working well
	ok, err := db.CheckKubeHealthy(re.PodLabels, re.Namespace, re.Service)
	if err != nil {
		return
	}
	re.IsHealthy = ok
	return
}

func GetDeployByRepo(repo *db.Repository) (re *DeployDes, err error) {
	defer func() {
		if err != nil && db.IsErrPipeNotFound(err) {
			re = &DeployDes{
				RepoID:   repo.ID,
				RepoName: repo.Name,
				ErrMsg:   err.Error(),
			}
		}
	}()
	repoID := repo.ID
	pipeline, err := db.GetLatestPipeline(repoID)
	if err != nil {
		return
	}
	if pipeline == nil {
		err = &db.ErrPipeNotFound{repoID, repo.Name}
		return
	}
	ptask, err := db.GetLatestPipeTask(pipeline.ID)
	if err != nil {
		return
	}
	if ptask == nil {
		err = &db.ErrPipeNotFound{repoID, repo.Name}
		return
	}

	return DescribePipeTask(pipeline, ptask, repo)
}
