// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package db

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"xorm.io/builder"
	"xorm.io/xorm"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/errutil"
)

var (
	ErrOrgNotExist = errors.New("Organization does not exist")
)

// IsOwnedBy returns true if given user is in the owner team.
func (u *User) IsOwnedBy(userID int64) bool {
	return IsOrganizationOwner(u.ID, userID)
}

// IsOrgMember returns true if given user is member of organization.
func (u *User) IsOrgMember(uid int64) bool {
	return u.IsOrganization() && IsOrganizationMember(u.ID, uid)
}

func (u *User) getTeam(e Engine, name string) (*Team, error) {
	return getTeamOfOrgByName(e, u.ID, name)
}

// GetTeamOfOrgByName returns named team of organization.
func (u *User) GetTeam(name string) (*Team, error) {
	return u.getTeam(x, name)
}

func (u *User) getOwnerTeam(e Engine) (*Team, error) {
	return u.getTeam(e, OWNER_TEAM)
}

// GetOwnerTeam returns owner team of organization.
func (u *User) GetOwnerTeam() (*Team, error) {
	return u.getOwnerTeam(x)
}

func (u *User) getTeams(e Engine) (err error) {
	u.Teams, err = getTeamsByOrgID(e, u.ID)
	return err
}

// GetTeams returns all teams that belong to organization.
func (u *User) GetTeams() error {
	return u.getTeams(x)
}

// TeamsHaveAccessToRepo returns all teamsthat have given access level to the repository.
func (u *User) TeamsHaveAccessToRepo(repoID int64, mode AccessMode) ([]*Team, error) {
	return GetTeamsHaveAccessToRepo(u.ID, repoID, mode)
}

// GetMembers returns all members of organization.
func (u *User) GetMembers(limit int) error {
	ous, err := GetOrgUsersByOrgID(u.ID, limit)
	if err != nil {
		return err
	}

	u.Members = make([]*User, len(ous))
	for i, ou := range ous {
		u.Members[i], err = GetUserByID(ou.Uid)
		if err != nil {
			return err
		}
	}
	return nil
}

// AddMember adds new member to organization.
func (u *User) AddMember(uid int64) error {
	return AddOrgUser(u.ID, uid)
}

// RemoveMember removes member from organization.
func (u *User) RemoveMember(uid int64) error {
	return RemoveOrgUser(u.ID, uid)
}

func (u *User) removeOrgRepo(e Engine, repoID int64) error {
	return removeOrgRepo(e, u.ID, repoID)
}

// RemoveOrgRepo removes all team-repository relations of organization.
func (u *User) RemoveOrgRepo(repoID int64) error {
	return u.removeOrgRepo(x, repoID)
}

// CreateOrganization creates record of a new organization.
func CreateOrganization(org, owner *User) (err error) {
	if err = isUsernameAllowed(org.Name); err != nil {
		return err
	}

	isExist, err := IsUserExist(0, org.Name)
	if err != nil {
		return err
	} else if isExist {
		return ErrUserAlreadyExist{args: errutil.Args{"name": org.Name}}
	}

	org.LowerName = strings.ToLower(org.Name)
	if org.Rands, err = GetUserSalt(); err != nil {
		return err
	}
	if org.Salt, err = GetUserSalt(); err != nil {
		return err
	}
	org.UseCustomAvatar = true
	org.MaxRepoCreation = -1
	org.NumTeams = 1
	org.NumMembers = 1

	sess := x.NewSession()
	defer sess.Close()
	if err = sess.Begin(); err != nil {
		return err
	}

	if _, err = sess.Insert(org); err != nil {
		return fmt.Errorf("insert organization: %v", err)
	}
	_ = org.GenerateRandomAvatar()

	// Add initial creator to organization and owner team.
	if _, err = sess.Insert(&OrgUser{
		Uid:      owner.ID,
		OrgID:    org.ID,
		IsOwner:  true,
		NumTeams: 1,
	}); err != nil {
		return fmt.Errorf("insert org-user relation: %v", err)
	}

	// Create default owner team.
	t := &Team{
		OrgID:      org.ID,
		LowerName:  strings.ToLower(OWNER_TEAM),
		Name:       OWNER_TEAM,
		Authorize:  AccessModeOwner,
		NumMembers: 1,
	}
	if _, err = sess.Insert(t); err != nil {
		return fmt.Errorf("insert owner team: %v", err)
	}

	if _, err = sess.Insert(&TeamUser{
		UID:    owner.ID,
		OrgID:  org.ID,
		TeamID: t.ID,
	}); err != nil {
		return fmt.Errorf("insert team-user relation: %v", err)
	}

	if err = os.MkdirAll(UserPath(org.Name), os.ModePerm); err != nil {
		return fmt.Errorf("create directory: %v", err)
	}

	return sess.Commit()
}

// GetOrgByName returns organization by given name.
func GetOrgByName(name string) (*User, error) {
	if len(name) == 0 {
		return nil, ErrOrgNotExist
	}
	u := &User{
		LowerName: strings.ToLower(name),
		Type:      UserOrganization,
	}
	has, err := x.Get(u)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, ErrOrgNotExist
	}
	return u, nil
}

// CountOrganizations returns number of organizations.
func CountOrganizations() int64 {
	count, _ := x.Where("type=1").Count(new(User))
	return count
}

// Organizations returns number of organizations in given page.
func Organizations(page, pageSize int) ([]*User, error) {
	orgs := make([]*User, 0, pageSize)
	return orgs, x.Limit(pageSize, (page-1)*pageSize).Where("type=1").Asc("id").Find(&orgs)
}

// DeleteOrganization completely and permanently deletes everything of organization.
func DeleteOrganization(org *User) (err error) {
	if err := DeleteUser(org); err != nil {
		return err
	}

	sess := x.NewSession()
	defer sess.Close()
	if err = sess.Begin(); err != nil {
		return err
	}

	if err = deleteBeans(sess,
		&Team{OrgID: org.ID},
		&OrgUser{OrgID: org.ID},
		&TeamUser{OrgID: org.ID},
	); err != nil {
		return fmt.Errorf("deleteBeans: %v", err)
	}

	if err = deleteUser(sess, org); err != nil {
		return fmt.Errorf("deleteUser: %v", err)
	}

	if err = sess.Commit(); err != nil {
		return err
	}
	return nil
}

// ExistOrgByExpStudent 检查是不是存在这样一个org，其实验ID是eid，而且还包含一个id为uid的用户
func ExistOrgByExpStudent(eid, uid int64) (bool, error) {
	return x.Table("user").
		Join("INNER", "org_user", "org_user.org_id = `user`.id").
		Where("org_user.uid = ? and `user`.exp_id = ?", uid, eid).
		Get(new(User))
}

// ________                ____ ___
// \_____  \_______  ____ |    |   \______ ___________
//  /   |   \_  __ \/ ___\|    |   /  ___// __ \_  __ \
// /    |    \  | \/ /_/  >    |  /\___ \\  ___/|  | \/
// \_______  /__|  \___  /|______//____  >\___  >__|
//         \/     /_____/              \/     \/

// OrgUser represents an organization-user relation.
type OrgUser struct {
	ID       int64
	Uid      int64 `xorm:"INDEX UNIQUE(s)"`
	OrgID    int64 `xorm:"INDEX UNIQUE(s)"`
	IsPublic bool
	IsOwner  bool
	NumTeams int
}

// IsOrganizationOwner returns true if given user is in the owner team.
func IsOrganizationOwner(orgID, userID int64) bool {
	has, _ := x.Where("is_owner = ?", true).And("uid = ?", userID).And("org_id = ?", orgID).Get(new(OrgUser))
	return has
}

// IsOrganizationMember returns true if given user is member of organization.
func IsOrganizationMember(orgId, uid int64) bool {
	has, _ := x.Where("uid=?", uid).And("org_id=?", orgId).Get(new(OrgUser))
	return has
}

// IsPublicMembership returns true if given user public his/her membership.
func IsPublicMembership(orgId, uid int64) bool {
	has, _ := x.Where("uid=?", uid).And("org_id=?", orgId).And("is_public=?", true).Get(new(OrgUser))
	return has
}

func getOrgsByUserID(sess *xorm.Session, userID int64, showAll bool) ([]*User, error) {
	orgs := make([]*User, 0, 10)
	if !showAll {
		sess.And("`org_user`.is_public=?", true)
	}
	return orgs, sess.And("`org_user`.uid=?", userID).
		Join("INNER", "`org_user`", "`org_user`.org_id=`user`.id").Find(&orgs)
}

// GetOrgsByUserID returns a list of organizations that the given user ID
// has joined.
func GetOrgsByUserID(userID int64, showAll bool) ([]*User, error) {
	return getOrgsByUserID(x.NewSession(), userID, showAll)
}

// GetOrgsByUserIDDesc returns a list of organizations that the given user ID
// has joined, ordered descending by the given condition.
func GetOrgsByUserIDDesc(userID int64, desc string, showAll bool) ([]*User, error) {
	return getOrgsByUserID(x.NewSession().Desc(desc), userID, showAll)
}

func getOwnedOrgsByUserID(sess *xorm.Session, userID int64) ([]*User, error) {
	orgs := make([]*User, 0, 10)
	return orgs, sess.Where("`org_user`.uid=?", userID).And("`org_user`.is_owner=?", true).
		Join("INNER", "`org_user`", "`org_user`.org_id=`user`.id").Find(&orgs)
}

// GetOwnedOrgsByUserID returns a list of organizations are owned by given user ID.
func GetOwnedOrgsByUserID(userID int64) ([]*User, error) {
	sess := x.NewSession()
	return getOwnedOrgsByUserID(sess, userID)
}

// GetOwnedOrganizationsByUserIDDesc returns a list of organizations are owned by
// given user ID, ordered descending by the given condition.
func GetOwnedOrgsByUserIDDesc(userID int64, desc string) ([]*User, error) {
	sess := x.NewSession()
	return getOwnedOrgsByUserID(sess.Desc(desc), userID)
}

// GetOrgIDsByUserID returns a list of organization IDs that user belongs to.
// The showPrivate indicates whether to include private memberships.
func GetOrgIDsByUserID(userID int64, showPrivate bool) ([]int64, error) {
	orgIDs := make([]int64, 0, 5)
	sess := x.Table("org_user").Where("uid = ?", userID)
	if !showPrivate {
		sess.And("is_public = ?", true)
	}
	return orgIDs, sess.Distinct("org_id").Find(&orgIDs)
}

func getOrgUsersByOrgID(e Engine, orgID int64, limit int) ([]*OrgUser, error) {
	orgUsers := make([]*OrgUser, 0, 10)

	sess := e.Where("org_id=?", orgID)
	if limit > 0 {
		sess = sess.Limit(limit)
	}
	return orgUsers, sess.Find(&orgUsers)
}

// GetOrgUsersByOrgID returns all organization-user relations by organization ID.
func GetOrgUsersByOrgID(orgID int64, limit int) ([]*OrgUser, error) {
	return getOrgUsersByOrgID(x, orgID, limit)
}

// ChangeOrgUserStatus changes public or private membership status.
func ChangeOrgUserStatus(orgID, uid int64, public bool) error {
	ou := new(OrgUser)
	has, err := x.Where("uid=?", uid).And("org_id=?", orgID).Get(ou)
	if err != nil {
		return err
	} else if !has {
		return nil
	}

	ou.IsPublic = public
	_, err = x.Id(ou.ID).AllCols().Update(ou)
	return err
}

func getUserOrg(orgID, uid int64) (*User, *User, error) {
	user, err := GetUserByID(uid)
	if err != nil {
		return nil, nil, err
	}

	org, err := GetUserByID(orgID)
	return user, org, err
}

// AddOrgUser adds new user to given organization.
func AddOrgUser(orgID, uid int64) error {
	if IsOrganizationMember(orgID, uid) {
		return nil
	}

	sess := x.NewSession()
	defer sess.Close()
	if err := sess.Begin(); err != nil {
		return err
	}

	ou := &OrgUser{
		Uid:   uid,
		OrgID: orgID,
	}

	if _, err := sess.Insert(ou); err != nil {
		return err
	} else if _, err = sess.Exec("UPDATE `user` SET num_members = num_members + 1 WHERE id = ?", orgID); err != nil {
		return err
	}

	if err := sess.Commit(); err != nil {
		return err
	}

	return nil
}

// RemoveOrgUser removes user from given organization.
func RemoveOrgUser(orgID, userID int64) error {
	ou := new(OrgUser)

	has, err := x.Where("uid=?", userID).And("org_id=?", orgID).Get(ou)
	if err != nil {
		return fmt.Errorf("get org-user: %v", err)
	} else if !has {
		return nil
	}

	user, err := GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("GetUserByID [%d]: %v", userID, err)
	}
	org, err := GetUserByID(orgID)
	if err != nil {
		return fmt.Errorf("GetUserByID [%d]: %v", orgID, err)
	}

	// FIXME: only need to get IDs here, not all fields of repository.
	repos, _, err := org.GetUserRepositories(user.ID, 1, org.NumRepos)
	if err != nil {
		return fmt.Errorf("GetUserRepositories [%d]: %v", user.ID, err)
	}

	// Check if the user to delete is the last member in owner team.
	if IsOrganizationOwner(orgID, userID) {
		t, err := org.GetOwnerTeam()
		if err != nil {
			return err
		}
		if t.NumMembers == 1 {
			return ErrLastOrgOwner{UID: userID}
		}
	}

	sess := x.NewSession()
	defer sess.Close()
	if err := sess.Begin(); err != nil {
		return err
	}

	if _, err := sess.ID(ou.ID).Delete(ou); err != nil {
		return err
	} else if _, err = sess.Exec("UPDATE `user` SET num_members=num_members-1 WHERE id=?", orgID); err != nil {
		return err
	}

	// Delete all repository accesses and unwatch them.
	repoIDs := make([]int64, len(repos))
	for i := range repos {
		repoIDs = append(repoIDs, repos[i].ID)
		if err = watchRepo(sess, user.ID, repos[i].ID, false); err != nil {
			return err
		}
	}

	if len(repoIDs) > 0 {
		if _, err = sess.Where("user_id = ?", user.ID).In("repo_id", repoIDs).Delete(new(Access)); err != nil {
			return err
		}
	}

	// Delete member in his/her teams.
	teams, err := getUserTeams(sess, org.ID, user.ID)
	if err != nil {
		return err
	}
	for _, t := range teams {
		if err = removeTeamMember(sess, org.ID, t.ID, user.ID); err != nil {
			return err
		}
	}

	if err := sess.Commit(); err != nil {
		return err
	}

	return nil
}

func removeOrgRepo(e Engine, orgID, repoID int64) error {
	_, err := e.Delete(&TeamRepo{
		OrgID:  orgID,
		RepoID: repoID,
	})
	return err
}

// RemoveOrgRepo removes all team-repository relations of given organization.
func RemoveOrgRepo(orgID, repoID int64) error {
	return removeOrgRepo(x, orgID, repoID)
}

func (u *User) getUserTeams(e Engine, userID int64, cols ...string) ([]*Team, error) {
	teams := make([]*Team, 0, u.NumTeams)
	return teams, e.Where("team_user.org_id = ?", u.ID).
		And("team_user.uid = ?", userID).
		Join("INNER", "team_user", "team_user.team_id = team.id").
		Cols(cols...).Find(&teams)
}

// GetUserTeamIDs returns of all team IDs of the organization that user is memeber of.
func (u *User) GetUserTeamIDs(userID int64) ([]int64, error) {
	teams, err := u.getUserTeams(x, userID, "team.id")
	if err != nil {
		return nil, fmt.Errorf("getUserTeams [%d]: %v", userID, err)
	}

	teamIDs := make([]int64, len(teams))
	for i := range teams {
		teamIDs[i] = teams[i].ID
	}
	return teamIDs, nil
}

// GetTeams returns all teams that belong to organization,
// and that the user has joined.
func (u *User) GetUserTeams(userID int64) ([]*Team, error) {
	return u.getUserTeams(x, userID)
}

// GetUserRepositories returns a range of repositories in organization which the user has access to,
// and total number of records based on given condition.
func (u *User) GetUserRepositories(userID int64, page, pageSize int) ([]*Repository, int64, error) {
	teamIDs, err := u.GetUserTeamIDs(userID)
	if err != nil {
		return nil, 0, fmt.Errorf("GetUserTeamIDs: %v", err)
	}
	if len(teamIDs) == 0 {
		// user has no team but "IN ()" is invalid SQL
		teamIDs = []int64{-1} // there is no team with id=-1
	}

	var teamRepoIDs []int64
	if err = x.Table("team_repo").In("team_id", teamIDs).Distinct("repo_id").Find(&teamRepoIDs); err != nil {
		return nil, 0, fmt.Errorf("get team repository IDs: %v", err)
	}
	if len(teamRepoIDs) == 0 {
		// team has no repo but "IN ()" is invalid SQL
		teamRepoIDs = []int64{-1} // there is no repo with id=-1
	}

	if page <= 0 {
		page = 1
	}
	repos := make([]*Repository, 0, pageSize)
	if err = x.Where("owner_id = ?", u.ID).
		And(builder.Or(
			builder.And(builder.Expr("is_private = ?", false), builder.Expr("is_unlisted = ?", false)),
			builder.In("id", teamRepoIDs))).
		Desc("updated_unix").
		Limit(pageSize, (page-1)*pageSize).
		Find(&repos); err != nil {
		return nil, 0, fmt.Errorf("get user repositories: %v", err)
	}

	repoCount, err := x.Where("owner_id = ?", u.ID).
		And(builder.Or(
			builder.Expr("is_private = ?", false),
			builder.In("id", teamRepoIDs))).
		Count(new(Repository))
	if err != nil {
		return nil, 0, fmt.Errorf("count user repositories: %v", err)
	}

	return repos, repoCount, nil
}

// GetUserMirrorRepositories returns mirror repositories of the organization which the user has access to.
func (u *User) GetUserMirrorRepositories(userID int64) ([]*Repository, error) {
	teamIDs, err := u.GetUserTeamIDs(userID)
	if err != nil {
		return nil, fmt.Errorf("GetUserTeamIDs: %v", err)
	}
	if len(teamIDs) == 0 {
		teamIDs = []int64{-1}
	}

	var teamRepoIDs []int64
	err = x.Table("team_repo").In("team_id", teamIDs).Distinct("repo_id").Find(&teamRepoIDs)
	if err != nil {
		return nil, fmt.Errorf("get team repository ids: %v", err)
	}
	if len(teamRepoIDs) == 0 {
		// team has no repo but "IN ()" is invalid SQL
		teamRepoIDs = []int64{-1} // there is no repo with id=-1
	}

	repos := make([]*Repository, 0, 10)
	if err = x.Where("owner_id = ?", u.ID).
		And("is_private = ?", false).
		Or(builder.In("id", teamRepoIDs)).
		And("is_mirror = ?", true). // Don't move up because it's an independent condition
		Desc("updated_unix").
		Find(&repos); err != nil {
		return nil, fmt.Errorf("get user repositories: %v", err)
	}
	return repos, nil
}
