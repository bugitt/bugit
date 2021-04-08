package project

import "git.scs.buaa.edu.cn/iobs/bugit/internal/db"

func GetUsersAllProjects(user *db.User) (projects, collaborativeProjects db.ProjectList, err error) {
	err = user.GetProjects(1, user.NumProjects)
	if err != nil {
		return
	}
	projects = user.Projects

	// 找到user参与的所有组织的所有仓库
	if len(user.Orgs) <= 0 {
		if err = user.GetOrganizations(true); err != nil {
			return
		}
	}
	collaborativeProjects = make(db.ProjectList, 0)
	var orgProjects db.ProjectList
	for _, org := range user.Orgs {
		orgProjects, err = db.GetUserAllProjects(org)
		if err != nil {
			return
		}
		collaborativeProjects = append(collaborativeProjects, orgProjects...)
	}
	err = collaborativeProjects.LoadAttributes()
	if err != nil {
		return
	}
	return
}
