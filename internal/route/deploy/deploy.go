package deploy

import (
	"fmt"
	"strconv"
	"strings"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/context"
	"git.scs.buaa.edu.cn/iobs/bugit/internal/db"
)

func CreatePost(c *context.Context) {
	projectID := c.QueryInt64("from")
	redirectPath := fmt.Sprintf("/project/%d", projectID)
	err := c.Req.ParseForm()
	if err != nil {
		c.Data["Err_Deploy"] = true
		c.RedirectSubpath(redirectPath)
	}

	repoIDs := make([]int64, 0)
	for k, v := range c.Req.Form {
		if strings.HasSuffix(k, "repo") {
			id, _ := strconv.ParseInt(v[0], 10, 64)
			repoIDs = append(repoIDs, id)
		}
	}

	for _, id := range repoIDs {
		// TODO: check error
		_ = db.CreateDeploy(&db.DeployOption{
			RepoID: id,
			Pusher: c.User,
		})
	}
	c.RedirectSubpath(redirectPath)
}
