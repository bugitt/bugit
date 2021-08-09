package db

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/conf"
	"git.scs.buaa.edu.cn/iobs/bugit/internal/tool"
	"github.com/artdarek/go-unzip"
	"github.com/bugitt/git-module"
	"github.com/loheagn/loclo/docker/container"
	"github.com/unknwon/com"
	"gopkg.in/yaml.v3"
	log "unknwon.dev/clog/v2"
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

type CIMeta struct {
}

type CIConfig struct {
	Version   string              `yaml:"version"`
	Meta      CIMeta              `yaml:"meta"`
	On        map[string][]string `yaml:"on"`
	PreBuild  []PreTaskConfig     `yaml:"pre_build"`
	Build     BuildTaskConfig     `yaml:"build"`
	PostBuild []PostTaskConfig    `yaml:"post_build"`
	Deploy    DeployTaskConfig    `yaml:"deploy"`
}

type PreTaskConfig struct {
	BaseTaskConf      `yaml:",inline"`
	Image             string `yaml:"image"`
	ContainerTaskConf `yaml:",inline"`
	CanSkip           bool `yaml:"can_skip"`
}

type BuildTaskConfig struct {
	BaseTaskConf `yaml:",inline"`
	DockerTag    string `yaml:"docker_tag"`
	Dockerfile   string `yaml:"dockerfile"`
	Scope        string `yaml:"scope"`
}

type PostTaskConfig struct {
	BaseTaskConf      `yaml:",inline"`
	ContainerTaskConf `yaml:",inline"`
	CanSkip           bool `yaml:"can_skip"`
}

type Port struct {
	Name     string `yaml:"name" json:"name"`
	Protocol string `yaml:"protocol" json:"protocol"`
	Port     int32  `yaml:"port" json:"port"`
}

type Cmd struct {
	Command []string `yaml:"command"`
	Args    []string `yaml:"args"`
}

type DeployTaskConfig struct {
	Envs       map[string]string `yaml:"envs"`
	Ports      []Port            `yaml:"ports"`
	Stateful   bool              `yaml:"stateful"`
	Storage    bool              `yaml:"storage"`
	WorkingDir string            `yaml:"workingDir"`
	Cmd        Cmd               `yaml:"cmd"`
}

type BaseTaskConf struct {
	Name     string `yaml:"name"`
	Describe string `yaml:"describe"`
	Type     string `yaml:"type"`
}

// ContainerTaskConf note: without imageTag !!!
type ContainerTaskConf struct {
	Env     map[string]string `yaml:"env"`
	WorkDir string            `yaml:"work_dir"`
	Cmd     []string          `yaml:"cmd"`
	Mount   map[string]string `yaml:"mount"`
}

func (c ContainerTaskConf) ToRunConf(ctxPath, imageTag string) *container.RunOption {
	mounts := make(map[string]string)
	for src, dist := range c.Mount {
		mounts[filepath.Join(ctxPath, src)] = dist
	}
	return &container.RunOption{
		Image:   imageTag,
		Cmd:     c.Cmd,
		Envs:    c.Env,
		WorkDir: c.WorkDir,
		Mounts:  mounts,
	}
}

func ParseCIConfig(input []byte) (*CIConfig, error) {
	ciConfig := &CIConfig{}
	err := yaml.Unmarshal(input, ciConfig)
	return ciConfig, err
}

func (c *CIConfig) ShouldCI(branch string, pipeType PipeType) bool {
	log.Trace("branch: %s", branch)
	for branchName, events := range c.On {
		if branchName == branch {
			for _, event := range events {
				if strings.ToLower(event) == strings.ToLower(string(pipeType)) {
					return true
				}
			}
			break
		}
	}
	return false
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
