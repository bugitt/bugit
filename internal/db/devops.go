package db

import (
	"github.com/bugitt/git-module"
	"gogs.io/gogs/internal/conf"
	"gogs.io/gogs/internal/db/errors"
	log "unknwon.dev/clog/v2"
)

func ciOnPush(commit *git.Commit) error {
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
		return errors.New("can not parse config")
	}

	ciConfig, err := ParseCIConfig(fileContent)
	if err != nil {
		return err
	}
	if !ciConfig.ShouldCIOnPush() {
		log.Info("no need for CI %s", commit.ID.String())
		return nil
	}

	return err
}
