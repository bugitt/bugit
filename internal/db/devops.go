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
