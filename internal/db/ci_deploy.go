package db

import (
	"fmt"
	"strconv"
	"time"

	jsoniter "github.com/json-iterator/go"
)

type DeployTask struct {
	SourceLog       string `xorm:"TEXT" json:"source_log"`
	IP              string
	Ports           []Port `xorm:"-" gorm:"-"`
	PortsS          string `xorm:"TEXT 'ports_s'" json:"ports_s"`
	NameSpace       string
	DeploymentName  string
	ServiceName     string
	BasicTaskResult `xorm:"extends"`
	BaseModel       `xorm:"extends"`
}

type GetCITaskDepDetail interface {
	IsSingle() bool
	IsSuccessful() bool
	GetDisplayName() string
	GetInternalName() string
	GetSourceLog() string
}

func (task *DeployTask) GetURLs() []string {
	urls := make([]string, len(task.Ports))
	for _, port := range task.Ports {
		urls = append(urls, fmt.Sprintf("%s:%d", task.IP, port.Port))
	}
	return urls
}

func (task *DeployTask) Run(ctx *CIContext) error {
	// Deploy
	err := Deploy(ctx, task)
	if err != nil {
		return err
	}
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
	task.IsSuccessful = true
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
	task.IsSuccessful = false
	task.EndUnix = time.Now().Unix()
	_, err := x.ID(task.ID).Cols("status", "is_succeed", "end_unix").Update(task)
	if err != nil {
		return err
	}
	_, err = x.ID(task.ID).Update(task)
	return err
}

func (task *DeployTask) StringPorts() {
	bytes, _ := jsoniter.Marshal(task.Ports)
	task.PortsS = string(bytes)
}

func (task *DeployTask) GetPorts() []Port {
	ports := make([]Port, 0)
	_ = jsoniter.Unmarshal([]byte(task.PortsS), &ports)
	task.Ports = ports
	return ports
}

func GetPodLabels(repo *Repository, branch, commit string) map[string]string {
	return map[string]string{
		"app":     repo.DeployName(),
		"project": strconv.FormatInt(repo.OwnerID, 10),
		"ref":     branch,
		"commit":  commit,
	}
}

func GetSvcLabels(repo *Repository) map[string]string {
	return map[string]string{
		"app":     repo.DeployName(),
		"project": strconv.FormatInt(repo.OwnerID, 10),
	}
}
