package devops

import (
	"github.com/artdarek/go-unzip"
	"github.com/bugitt/git-module"
	"github.com/unknwon/com"
	"gogs.io/gogs/internal/conf"
	"gogs.io/gogs/internal/db"
	"gogs.io/gogs/internal/tool"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

type CIConfig struct {
	Version    string      `yaml:"version"`
	Meta       Meta        `yaml:"meta"`
	Validation []ValidTask `yaml:"validation"`
	Build      []BaseTask  `yaml:"build"`
}

type Meta struct {
	Tag string `yaml:"tag"`
}

type BaseTask struct {
	Name     string `yaml:"name"`
	Describe string `yaml:"describe"`
	Type     string `yaml:"type"`
}

type Threshold struct {
	Warning int `yaml:"warning"`
	Error   int `yaml:"error"`
}

type ValidTask struct {
	BaseTask
	Scope     []string  `yaml:"scope"`
	Threshold Threshold `yaml:"threshold"`
}

type BuildTask struct {
	BaseTask
	Dockerfile string `yaml:"dockerfile"`
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
	gitRepo, err := git.Open(db.RepoPath(ownerName, repoName))
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
