package org

import (
	"git.scs.buaa.edu.cn/iobs/bugit/internal/context"
	"git.scs.buaa.edu.cn/iobs/bugit/internal/db"
)

type Member struct {
	StudentID string `json:"student_id"`
	Email     string `json:"email"`
}

func ListMembers(c *context.APIContext) {
	org := c.Org.Organization
	orgUsers, err := db.GetOrgUsersByOrgID(org.ID, -1)
	if err != nil {
		c.Error(err, "get orgUsers")
		return
	}
	users := make([]*Member, 0, len(orgUsers))
	for _, ou := range orgUsers {
		u, err := db.GetUserByID(ou.Uid)
		if err != nil {
			c.Error(err, "get user by id")
			return
		}
		users = append(users, &Member{
			StudentID: u.StudentID,
			Email:     u.Email,
		})
	}
	c.JSONSuccess(users)
}
