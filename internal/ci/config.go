package ci

import (
	"strings"

	log "unknwon.dev/clog/v2"
)

type Config struct {
	Version string `yaml:"version"`
	//Meta     Meta                `yaml:"meta"`
	//On       map[string][]string `yaml:"on"`
	//Validate []ValidTaskConfig   `yaml:"validate"`
	//Build    BuildTaskConfig     `yaml:"build"`
	//Test     []TestTaskConfig    `yaml:"test"`
	//Deploy   DeployTaskConfig    `yaml:"deploy"`
}

func (c *Config) ShouldCIOnPush(refName string) bool {
	log.Trace("refName: %s", refName)
	var events []string
	for k, v := range c.On {
		if k == refName {
			events = v
			break
		}
	}
	for _, s := range events {
		if strings.ToLower(s) == DevopsPush {
			return true
		}
	}
	return false
}

func (c *Config) ShouldCIOnPR(refName string) bool {
	var events []string
	for k, v := range c.On {
		if k == refName {
			events = v
			break
		}
	}
	for _, s := range events {
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
