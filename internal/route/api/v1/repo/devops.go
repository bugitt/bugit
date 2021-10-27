package repo

import (
	"net/http"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/ci"
	"git.scs.buaa.edu.cn/iobs/bugit/internal/context"
)

type CreatePipelineOption struct {
	Branch string `json:"branch"`
	Commit string `json:"commit"`
}

func CreatePipeline(c *context.APIContext, form CreatePipelineOption) {
	err := ci.CreatePipeline(&ci.CreatePipelineOption{
		GitRepo: c.Repo.GitRepo,
		Repo:    c.Repo.Repository,
		Pusher:  c.User,
		Branch:  form.Branch,
		Commit:  form.Commit,
	})
	if err == ci.ErrCreatePipelineCIFileInvalid {
		c.JSON(http.StatusBadRequest, "no invalid ci file")
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, "internal server error")
		return
	}
	c.JSONSuccess("push ci task to queue successfully")
}
