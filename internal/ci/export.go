package ci

import (
	"context"
	"fmt"
	"strconv"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/db"
	"git.scs.buaa.edu.cn/iobs/bugit/internal/kube"
	"github.com/bugitt/git-module"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
)

type GetPipelinesOption struct {
	Repo   *db.Repository
	Branch string
	Page   int
	Size   int
}

type PipelineDes struct {
	BranchURL    string
	CommitURL    string
	PrettyCommit string
	Pusher       *db.User
	Pipeline     *db.Pipeline
	PreBuild     []*db.PreBuildResult
	Build        *db.BuildResult
	PostBuild    []*db.PostBuildResult
	Push         []*db.PushResult
}

// DeployDes 描述一个project中的一个仓库最新的部署情况
type DeployDes struct {
	Exist bool
	// 总体描述
	Branch       string
	BranchURL    string
	Commit       string
	CommitURL    string
	PrettyCommit string
	Pusher       *db.User

	// 容器情况
	PodStatus  v1.PodPhase
	PodHealthy bool
	PodURL     string

	// 服务情况
	SvcHealthy bool
	IP         string
	Ports      []kube.SvcPort
	SvcURL     string
}

type CreatePipelineOption struct {
	GitRepo *git.Repository
	Repo    *db.Repository
	Pusher  *db.User
	Branch  string
	Commit  string
}

func GetDeployDes(ctx context.Context, repo *db.Repository) (*DeployDes, error) {
	namespace := repo.Owner.KSProjectName
	deployName := repo.LowerName
	cli, err := kube.NewClient(ctx, &kube.CreateClientOpt{Namespace: namespace})
	if err != nil {
		return nil, err
	}

	// 尝试获取相对比较健康的Pod
	getHealthyPod := func() (v1.Pod, error) {
		pods, err := cli.GetPodsByLabels(ctx, map[string]string{"repo": repo.LowerName})
		if err != nil {
			return v1.Pod{}, err
		}
		result := v1.Pod{}
		setFlag := false
		for _, p := range pods {
			if phase := p.Status.Phase; phase == v1.PodRunning || phase == v1.PodSucceeded {
				return p, nil
			} else if phase == v1.PodPending {
				result = p
			} else {
				if !setFlag {
					result = p
					setFlag = true
				}
			}
		}
		return result, nil
	}
	pod, err := getHealthyPod()
	if err != nil && !kerrors.IsNotFound(err) {
		return nil, err
	}
	if kerrors.IsNotFound(err) {
		return &DeployDes{Exist: false}, nil
	}
	podHealthy := pod.Status.Phase == v1.PodRunning || pod.Status.Phase == v1.PodSucceeded

	pipelineID, _ := strconv.ParseInt(pod.Labels["pipelineID"], 10, 64)
	pipeline, err := db.GetPipelineByID(pipelineID)
	if err != nil {
		return nil, err
	}
	pusher, err := pipeline.GetPusher()
	if err != nil {
		return nil, err
	}

	deploy := &DeployDes{
		Exist: true,

		Branch:       pipeline.RefName,
		BranchURL:    fmt.Sprintf("%s/src/%s", repo.Link(), pipeline.RefName),
		Commit:       pipeline.Commit,
		CommitURL:    fmt.Sprintf("%s/commit/%s", repo.Link(), pipeline.Commit),
		PrettyCommit: pipeline.Commit[:10],
		Pusher:       pusher,

		PodStatus:  pod.Status.Phase,
		PodHealthy: podHealthy,
		PodURL:     fmt.Sprintf("https://kube.scs.buaa.edu.cn/main-workspace/clusters/default/projects/%s/deployments/%s/resource-status", namespace, deployName),
	}

	svc, err := cli.GetService(ctx, deployName)
	if err != nil && !kerrors.IsNotFound(err) {
		return nil, err
	}
	svcHealthy := false
	if kerrors.IsNotFound(err) {
		svcHealthy = false
	} else {
		svcHealthy = true
	}
	deploy.SvcHealthy = svcHealthy
	if svc == nil {
		return deploy, nil
	}
	ports := make([]kube.SvcPort, 0, len(svc.Spec.Ports))
	for _, port := range svc.Spec.Ports {
		ports = append(ports, kube.SvcPort{
			Port: kube.Port{
				Name:     port.Name,
				Protocol: string(port.Protocol),
				Port:     port.TargetPort.IntVal,
			},
			ExportPort: port.NodePort,
		})
	}
	deploy.IP = NextIP()
	deploy.Ports = ports
	deploy.SvcURL = fmt.Sprintf("https://kube.scs.buaa.edu.cn/main-workspace/clusters/default/projects/%s/services/%s/resource-status", namespace, deployName)

	return deploy, nil
}

func GetPipelineDesList(opt *GetPipelinesOption) ([]*PipelineDes, error) {
	pipes, err := db.GetPipelineByRepoPage(opt.Repo.ID, opt.Branch, opt.Page, opt.Size)
	if err != nil {
		return nil, err
	}
	result := make([]*PipelineDes, 0, len(pipes))
	for _, p := range pipes {
		pd := &PipelineDes{
			BranchURL:    fmt.Sprintf("%s/src/%s", opt.Repo.Link(), p.RefName),
			CommitURL:    fmt.Sprintf("%s/commit/%s", opt.Repo.Link(), p.Commit),
			PrettyCommit: p.Commit[:10],
			Pipeline:     p,
		}
		pusher, err := p.GetPusher()
		if err != nil {
			return nil, err
		}
		pd.Pusher = pusher

		pres, err := p.GetPreBuildResult()
		if err != nil {
			return nil, err
		}
		pd.PreBuild = pres

		build, err := p.GetBuildResult()
		if err != nil {
			return nil, err
		}
		pd.Build = build

		posts, err := p.GetPostBuildResult()
		if err != nil {
			return nil, err
		}
		pd.PostBuild = posts

		pushs, err := p.GetPushResult()
		if err != nil {
			return nil, err
		}
		pd.Push = pushs

		result = append(result, pd)
	}

	return result, nil
}

func CreatePipeline(opt *CreatePipelineOption) error {
	if opt.Repo == nil || opt.GitRepo == nil {
		return ErrCreatePipelineParam
	}
	repo := opt.Repo
	if len(opt.Branch) <= 0 {
		opt.Branch = repo.DefaultBranch
	}
	if len(opt.Commit) <= 0 {
		commit, err := opt.GitRepo.BranchCommit(opt.Branch)
		if err != nil {
			return err
		}
		opt.Commit = commit.ID.String()
	}
	gitCommit, err := opt.GitRepo.CatFileCommit(opt.Commit)
	if err != nil {
		return err
	}

	// 检查对应的commit中是否有合法的 CIConfig 配置文件
	ciConfig, err := db.GetCIConfigFromCommit(gitCommit)
	if err != nil || ciConfig == nil {
		return ErrCreatePipelineCIFileInvalid
	}

	// 触发新的部署
	_, err = db.PreparePipeline(gitCommit, db.MANUAL, opt.Repo, opt.Pusher, opt.Branch, ciConfig, nil)
	if err != nil {
		return err
	}

	go Queue.Add(repo.ID)
	return nil
}
