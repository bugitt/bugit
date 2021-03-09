package db

import (
	"github.com/bugitt/git-module"
	"gogs.io/gogs/internal/conf"
	"gogs.io/gogs/internal/db/errors"
	"gogs.io/gogs/internal/sync"
	log "unknwon.dev/clog/v2"
)

var CIQueue = sync.NewUniqueQueue(1000)

func shouldCIOnPush(commit *git.Commit) (bool, error) {
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
	return ciConfig.ShouldCIOnPush(), nil
}

func CI() {
	for t := range CIQueue.Queue() {
		log.Trace("ciTest: %s", t)
	}
}
