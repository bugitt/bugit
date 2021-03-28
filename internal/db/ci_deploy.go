package db

import "time"

// Port
type Port struct {
	Name     string `yaml:"name"`
	Protocol string `yaml:"protocol"`
	Port     int    `yaml:"port"`
}

// Cmd
type Cmd struct {
	Command []string `yaml:"command"`
	Args    []string `yaml:"args"`
}

// DeployTaskConfig
type DeployTaskConfig struct {
	Envs       map[string]interface{} `yaml:"envs"`
	Ports      []Port                 `yaml:"ports"`
	Stateful   bool                   `yaml:"stateful"`
	Storage    bool                   `yaml:"storage"`
	WorkingDir string                 `yaml:"workingDir"`
	Cmd        Cmd                    `yaml:"cmd"`
}

type DeployTask struct {
	SourceLog string `xorm:"TEXT" json:"source_log"`
	Url       string
	Port      int
	BasicTask `xorm:"extends"`
	BaseModel `xorm:"extends"`
}

func (task *DeployTask) Run(ctx *CIContext) error {
	return nil
}

func (task *DeployTask) start() error {
	task.Status = Running
	task.BeginUnix = time.Now().Unix()
	_, err := x.ID(task.ID).Cols("status", "is_started", "begin_unix").Update(task)
	return err
}

func (task *DeployTask) success() error {
	task.Status = Finished
	task.IsSucceed = true
	task.EndUnix = time.Now().Unix()
	_, err := x.ID(task.ID).Cols("status", "is_succeed", "end_unix").Update(task)
	if err != nil {
		return err
	}
	_, err = x.ID(task.ID).Update(task)
	return err
}

func (task *DeployTask) failed() error {
	task.Status = Finished
	task.IsSucceed = false
	task.EndUnix = time.Now().Unix()
	_, err := x.ID(task.ID).Cols("status", "is_succeed", "end_unix").Update(task)
	if err != nil {
		return err
	}
	_, err = x.ID(task.ID).Update(task)
	return err
}
