package ci

import (
	"fmt"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/db"
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
