package db

import (
	"github.com/bugitt/git-module"
	"gogs.io/gogs/internal/conf"
	"gogs.io/gogs/internal/db/errors"
	"gogs.io/gogs/internal/sync"
	log "unknwon.dev/clog/v2"
)

var CIQueue = sync.NewUniqueQueue(1000)

func shouldCIOnPush(commit *git.Commit, repo *Repository, pusher *User, refName string) (bool, error) {
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
		// TODO: 无法解析config的时候是不是要给用户提示？
		return false, errors.New("can not parse config")
	}

	ciConfig, err := ParseCIConfig(fileContent)
	if err != nil {
		return false, err
	}
	log.Trace("%#v", ciConfig)
	shouldCI := ciConfig.ShouldCIOnPush()
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

func ci() {
	for t := range CIQueue.Queue() {
		log.Trace("ciTest: %s", t)
	}
}

func StartCI() {
	go ci()
}
