package db

import (
	"fmt"

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

	config, err := ParseCIConfig(fileContent)
	if err != nil {
		_, err := PreparePipeline(commit, PUSH, repo, pusher, refName, config, err)
		if err != nil {
			log.Error("%s", err.Error())
			return false, err
		}
	}

	if !config.ShouldCI(refName, PUSH) {
		return false, nil
	}

	// 创建 pipeline
	pipeline, err := PreparePipeline(commit, PUSH, repo, pusher, refName, config, err)
	if err != nil {
		log.Error("%s", err.Error())
		return false, err
	}

	log.Trace("%d", pipeline.ID)

	return true, nil
}

func genImageTag(repo *Repository, commit string) string {
	_ = repo.LoadAttributes()
	return fmt.Sprintf("%s/%s/%s:%s", conf.Docker.Registry, repo.Owner.HarborName, repo.LowerName, commit)
}

func GetCIConfigFromCommit(commit *git.Commit) (*CIConfig, error) {
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
		return nil, nil
	}
	ciConfig, err := ParseCIConfig(fileContent)
	if err != nil {
		return nil, err
	}
	return ciConfig, nil
}
