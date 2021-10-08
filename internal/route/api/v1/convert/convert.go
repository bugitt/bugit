// Copyright 2015 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package convert

import (
	"fmt"

	"github.com/unknwon/com"

	"github.com/bugitt/git-module"
	api "github.com/gogs/go-gogs-client"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/db"
)

type Organization struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	ExpName     string `json:"exp_name"`
	BugitURL    string `json:"bugit_url"`
	KSURL       string `json:"ks_url"`
	HarborURL   string `json:"harbor_url"`
	CourseName  string `json:"course_name"`
	AvatarUrl   string `json:"avatar_url"`
	Description string `json:"description"`
	Website     string `json:"website"`
	Location    string `json:"location"`
}

func ToEmail(email *db.EmailAddress) *api.Email {
	return &api.Email{
		Email:    email.Email,
		Verified: email.IsActivated,
		Primary:  email.IsPrimary,
	}
}

func ToBranch(b *db.Branch, c *git.Commit) *api.Branch {
	return &api.Branch{
		Name:   b.Name,
		Commit: ToCommit(c),
	}
}

func ToCommit(c *git.Commit) *api.PayloadCommit {
	authorUsername := ""
	author, err := db.GetUserByEmail(c.Author.Email)
	if err == nil {
		authorUsername = author.Name
	}
	committerUsername := ""
	committer, err := db.GetUserByEmail(c.Committer.Email)
	if err == nil {
		committerUsername = committer.Name
	}
	return &api.PayloadCommit{
		ID:      c.ID.String(),
		Message: c.Message,
		URL:     "Not implemented",
		Author: &api.PayloadUser{
			Name:     c.Author.Name,
			Email:    c.Author.Email,
			UserName: authorUsername,
		},
		Committer: &api.PayloadUser{
			Name:     c.Committer.Name,
			Email:    c.Committer.Email,
			UserName: committerUsername,
		},
		Timestamp: c.Author.When,
	}
}

func ToPublicKey(apiLink string, key *db.PublicKey) *api.PublicKey {
	return &api.PublicKey{
		ID:      key.ID,
		Key:     key.Content,
		URL:     apiLink + com.ToStr(key.ID),
		Title:   key.Name,
		Created: key.Created,
	}
}

func ToHook(repoLink string, w *db.Webhook) *api.Hook {
	config := map[string]string{
		"url":          w.URL,
		"content_type": w.ContentType.Name(),
	}
	if w.HookTaskType == db.SLACK {
		s := w.SlackMeta()
		config["channel"] = s.Channel
		config["username"] = s.Username
		config["icon_url"] = s.IconURL
		config["color"] = s.Color
	}

	return &api.Hook{
		ID:      w.ID,
		Type:    w.HookTaskType.Name(),
		URL:     fmt.Sprintf("%s/settings/hooks/%d", repoLink, w.ID),
		Active:  w.IsActive,
		Config:  config,
		Events:  w.EventsArray(),
		Updated: w.Updated,
		Created: w.Created,
	}
}

func ToDeployKey(apiLink string, key *db.DeployKey) *api.DeployKey {
	return &api.DeployKey{
		ID:       key.ID,
		Key:      key.Content,
		URL:      apiLink + com.ToStr(key.ID),
		Title:    key.Name,
		Created:  key.Created,
		ReadOnly: true, // All deploy keys are read-only.
	}
}

func ToOrganization(org *db.User) *Organization {
	return &Organization{
		ID:          org.ID,
		AvatarUrl:   org.AvatarLink(),
		Name:        org.Name,
		FullName:    org.FullName,
		BugitURL: "https://git.scs.buaa.edu.cn/"+org.Name,
		KSURL: fmt.Sprintf("https://kube.scs.buaa.edu.cn/main-workspace/clusters/default/projects/%s/overview", org.KSProjectName),
		HarborURL: fmt.Sprintf("https://harbor.scs.buaa.edu.cn/harbor/projects/%d/repositories", org.HarborProjectID),
		Description: org.Description,
		Website:     org.Website,
		Location:    org.Location,
		ExpName:     org.ExpName,
		CourseName:  org.CourseName,
	}
}

func ToTeam(team *db.Team) *api.Team {
	return &api.Team{
		ID:          team.ID,
		Name:        team.Name,
		Description: team.Description,
		Permission:  team.Authorize.String(),
	}
}
