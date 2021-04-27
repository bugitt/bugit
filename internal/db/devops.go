package db

import (
	"git.scs.buaa.edu.cn/iobs/bugit/internal/conf"
	"git.scs.buaa.edu.cn/iobs/bugit/internal/db/errors"
	"git.scs.buaa.edu.cn/iobs/bugit/internal/sync"
	"github.com/bugitt/git-module"
	log "unknwon.dev/clog/v2"
)

var CIQueue = sync.NewUniqueQueue(1000)

func shouldCIOnPush(commit *git.Commit, repo *Repository, pusher *User, refName string) (bool, error) {
	ciConfig, fileContent, err := getCIConfigFromCommit(commit)
	if err != nil {
		return false, err
	}
	if ciConfig == nil {
		// TODO: 无法解析config的时候是不是要给用户提示？
		return false, errors.New("can not parse config")
	}

	shouldCI := ciConfig.ShouldCIOnPush(refName)
	if !shouldCI {
		return false, nil
	}

	// 创建 pipeline
	pipeline, err := preparePipeline(commit, fileContent, repo, pusher, refName)
	if err != nil {
		log.Error("%s", err.Error())
		return false, err
	}

	log.Trace("%d", pipeline.ID)

	// 创建 pipelineTask
	if err := preparePipeTask(pipeline, pusher); err != nil {
		return false, err
	}

	go CIQueue.Add(repo.ID)
	return true, nil
}

func getCIConfigFromCommit(commit *git.Commit) (*CIConfig, []byte, error) {
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
	ciConfig, err := ParseCIConfig(fileContent)
	if err != nil {
		return nil, nil, err
	}
	return ciConfig, fileContent, nil
}

func ci() {
	tasks := make([]*PipeTask, 0, 5)
	if err := x.Where("stage = ?", NotStart).Find(&tasks); err != nil {
		log.Error("Get pre pipe tasks: %v", err)
	}
	for _, ptask := range tasks {
		go ptask.Run()

	}
	for repoID := range CIQueue.Queue() {
		log.Trace("Begin Pipeline for [repo_id: %v]", repoID)
		CIQueue.Remove(repoID)

		tasks := make([]*PipeTask, 0, 5)
		if err := x.Where("repo_id = ?", repoID).And("stage = ?", NotStart).Find(&tasks); err != nil {
			log.Error("Get repository [%s] pipe tasks: %v", repoID, err)
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
	Repo      *Repository     `json:"-"`
	gitRepo   *git.Repository `json:"-"`
	Branch    string          `json:"Branch" form:"Branch"`
	Commit    string          `json:"Commit" form:"Commit"`
	gitCommit *git.Commit     `json:"-"`
	Pusher    *User           `json:"-"`
}

func CreateDeploy(opt *DeployOption) (err error) {
	// 1. 完善参数
	if opt.Repo != nil {
		opt.RepoID = opt.Repo.ID
	} else {
		repo, err := GetRepositoryByID(opt.RepoID)
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
		return &ErrNoValidCIConfig{
			RepoName: opt.Repo.Name,
			Branch:   opt.Branch,
			Commit:   opt.Commit,
		}
	}

	// 3. 首先检查是不是真的需要进行一次 deploy
	//    当前的计划是，如果当前有同样commit的deploy正在执行，那么就忽略本次部署请求
	isRunning, err := IsPipelineRunning(opt.RepoID, opt.Commit)
	if err != nil {
		return err
	}
	if isRunning {
		return &ErrNoNeedDeploy{"the deployment of the current commit is in progress, please stay calm"}
	}

	// 4. 好了，终于确定了，可以触发新的部署了
	pipeline, err := preparePipeline(opt.gitCommit, fileContent, opt.Repo, opt.Pusher, opt.Branch)
	if err != nil {
		return err
	}
	if err := preparePipeTask(pipeline, opt.Pusher); err != nil {
		return err
	}

	go CIQueue.Add(opt.Repo.ID)
	return nil
}
