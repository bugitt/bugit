package db

import (
	"git.scs.buaa.edu.cn/iobs/bugit/internal/conf"
	"github.com/bugitt/git-module"
	log "unknwon.dev/clog/v2"
)

// CheckCIFile 检查本次commit中有没有指定的CI配置文件
func CheckCIFile(commit *git.Commit) (bool, string, error) {
	for _, filename := range conf.Devops.Filename {
		if ok, err := commit.ExistsFile(filename); err != nil {
			return false, "", err
		} else if ok {
			return true, filename, err
		}
	}
	return false, "", nil
}

func shouldCIOnPush(commit *git.Commit, repo *Repository, pusher *User, refName string) (bool, error) {
	exist, filename, err := CheckCIFile(commit)
	if err != nil {
		return false, err
	}
	if !exist {
		return false, nil
	}

	fileContent, err := commit.ReadFileSimple(filename)
	if err != nil {
		return false, err
	}

	// 创建 pipeline
	pipeline, err := preparePipeline(commit, fileContent, repo, pusher, refName)
	if err != nil {
		log.Error("%s", err.Error())
		return false, err
	}

	log.Trace("%d", pipeline.ID)

	return true, nil
}
