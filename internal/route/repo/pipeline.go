package repo

import (
	cc "context"
	"sort"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/ci"
	"git.scs.buaa.edu.cn/iobs/bugit/internal/context"
)

const PIPELINES = "repo/pipeline/pipeline"

func Pipelines(c *context.Context) {
	c.Data["PageIsPipelineList"] = true

	pipelineDesList, err := ci.GetPipelineDesList(&ci.GetPipelinesOption{
		Repo: c.Repo.Repository,
		Page: 1,
		Size: 1000,
	})
	if err != nil {
		c.Error(err, "get pipeline list")
		return
	}
	sort.Slice(pipelineDesList, func(i, j int) bool {
		return pipelineDesList[i].Pipeline.BeginUnix > pipelineDesList[j].Pipeline.BeginUnix
	})
	for _, p := range pipelineDesList {
		prettyLog(p)
	}

	deployDes, err := ci.GetDeployDes(cc.Background(), c.Repo.Repository)
	if err != nil {
		c.Error(err, "get deploy")
		return
	}

	c.Data["PipelineDesList"] = pipelineDesList
	c.Data["DeployDes"] = deployDes
	c.Data["ExistDeploy"] = deployDes.Exist

	c.Success(PIPELINES)
}

func prettyLog(p *ci.PipelineDes) {
	for _, p := range p.PreBuild {
		p.ConvertLogHTML()
	}
	p.Build.ConvertLogHTML()
	for _, p := range p.PostBuild {
		p.ConvertLogHTML()
	}
	for _, p := range p.Push {
		p.ConvertLogHTML()
	}
}
