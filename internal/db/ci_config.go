package db

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/artdarek/go-unzip"
	"github.com/bugitt/git-module"
	"github.com/unknwon/com"
	"gogs.io/gogs/internal/conf"
	"gogs.io/gogs/internal/tool"
	"gopkg.in/yaml.v3"
)

const (
	DevopsPush string = "push"
	DevopsPR   string = "pr"
	DevopsMR   string = "mr"
)

type ErrConfFileNotFound struct {
	repoPath string
}

func (err ErrConfFileNotFound) Error() string {
	return fmt.Sprintf("devops conf file not found: repo: %s", err.repoPath)
}

func IsErrConfFileNotFound(err error) bool {
	_, ok := err.(ErrConfFileNotFound)
	return ok
}

type CIConfig struct {
	Version    string            `yaml:"version"`
	Meta       Meta              `yaml:"meta"`
	On         []string          `yaml:"on"`
	Validation []ValidTaskConfig `yaml:"validation"`
	Build      []BuildTaskConfig `yaml:"build"`
}

type Meta struct {
	Tag string `yaml:"tag"`
}

type BaseTaskConfig struct {
	Name     string `yaml:"name"`
	Describe string `yaml:"describe"`
	Type     string `yaml:"type"`
}

type BuildTaskConfig struct {
	BaseTaskConfig `yaml:",inline"`
	Dockerfile     string `yaml:"dockerfile"`
	Scope          string `yaml:"scope"`
}

func (c *CIConfig) ShouldCIOnPush() bool {
	for _, s := range c.On {
		if strings.ToLower(s) == DevopsPush {
			return true
		}
	}
	return false
}

func (c *CIConfig) ShouldCIOnPR() bool {
	for _, s := range c.On {
		lowS := strings.ToLower(s)
		if lowS == DevopsPR ||
			lowS == "pull request" ||
			lowS == "merge request" ||
			lowS == DevopsMR {
			return true
		}
	}
	return false
}

func ParseCIConfig(input []byte) (*CIConfig, error) {
	ciConfig := &CIConfig{}
	err := yaml.Unmarshal(input, ciConfig)
	return ciConfig, err
}

func ReadConf(ownerName, repoName, refName string) (*CIConfig, error) {
	repoPath, err := loadRepo(ownerName, repoName, refName)
	if err != nil {
		return nil, err
	}

	var confPath string
	for _, name := range conf.Devops.Filename {
		confPath = filepath.Join(repoPath, name)
		if com.IsFile(confPath) {
			break
		}
		confPath = ""
	}
	if len(confPath) <= 0 {
		return nil, ErrConfFileNotFound{repoPath: repoPath}
	}

	ciConfig := &CIConfig{}
	data, err := ioutil.ReadFile(confPath)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(data, ciConfig)
	return ciConfig, err
}

func loadRepo(ownerName, repoName, refName string) (repoPath string, err error) {
	gitRepo, err := git.Open(RepoPath(ownerName, repoName))
	if err != nil {
		return
	}
	var commit *git.Commit
	if gitRepo.HasBranch(refName) {
		commit, err = gitRepo.BranchCommit(refName)
		if err != nil {
			return
		}
	} else if gitRepo.HasTag(refName) {
		commit, err = gitRepo.TagCommit(refName)
		if err != nil {
			return
		}
	} else if len(refName) >= 7 && len(refName) <= 40 {
		commit, err = gitRepo.CatFileCommit(refName)
		if err != nil {
			return
		}
	} else {
		return
	}
	hash := tool.ShortSHA1(commit.ID.String())
	archivePath := filepath.Join(gitRepo.Path(), "archives", "zip")
	if !com.IsDir(archivePath) {
		if err = os.MkdirAll(archivePath, os.ModePerm); err != nil {
			return
		}
	}
	archiveFormat := git.ArchiveZip
	archivePath = path.Join(archivePath, hash+".zip")
	if !com.IsFile(archivePath) {
		if err = commit.CreateArchive(archiveFormat, archivePath); err != nil {
			return
		}
	}

	repoPath = filepath.Join(conf.Devops.Tmpdir, ownerName, repoName, hash)
	if !com.IsDir(repoPath) {
		uz := unzip.New(archivePath, repoPath)
		err = uz.Extract()
		if err != nil {
			return
		}
	}
	return
}
