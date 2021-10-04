package platform

import (
	"context"
	"strconv"
	"strings"

	"github.com/go-openapi/runtime"

	httptransport "github.com/go-openapi/runtime/client"
	hclient "github.com/loheagn/harbor-client/client"
	hmember "github.com/loheagn/harbor-client/client/member"
	hproject "github.com/loheagn/harbor-client/client/project"
	hrepo "github.com/loheagn/harbor-client/client/repository"
	huser "github.com/loheagn/harbor-client/client/user"
	hmodels "github.com/loheagn/harbor-client/models"
)

type HarborCli struct {
	api      *hclient.HarborAPI
	authInfo runtime.ClientAuthInfoWriter
}

func NewHarborCli(host, adminName, adminPassword string) *HarborCli {
	if len(host) <= 0 {
		host = "harbor.scs.buaa.edu.cn"
	}

	api := hclient.New(httptransport.New(
		host,
		hclient.DefaultBasePath,
		hclient.DefaultSchemes,
	), nil)
	authInfo := httptransport.BasicAuth(adminName, adminPassword)

	return &HarborCli{
		api:      api,
		authInfo: authInfo,
	}
}

func (cli HarborCli) searchUser(ctx context.Context, studentID string) ([]*hmodels.UserSearchRespItem, error) {
	studentID = strings.ToLower(studentID)
	respOK, err := cli.api.User.SearchUsers(&huser.SearchUsersParams{
		Context:  ctx,
		Username: studentID,
	}, cli.authInfo)
	if err != nil {
		return nil, err
	}

	return respOK.GetPayload(), nil
}

func (cli HarborCli) ExistUser(ctx context.Context, studentID string) (bool, error) {
	resp, err := cli.searchUser(ctx, studentID)
	if err != nil {
		return false, err
	}
	return len(resp) > 0, nil
}

func (cli HarborCli) GetUser(ctx context.Context, studentID string) (*User, error) {
	resp, err := cli.searchUser(ctx, studentID)
	if err != nil {
		return nil, err
	}
	if len(resp) <= 0 {
		return nil, ErrNotFound
	}
	u := resp[0]
	if u.Username != studentID {
		return nil, ErrNotFound
	}
	return &User{
		IntID: u.UserID,
		Name:  u.Username,
	}, nil
}

func (cli HarborCli) DeleteUser(ctx context.Context, u *User) error {
	_, err := cli.api.User.DeleteUser(&huser.DeleteUserParams{
		UserID:  u.IntID,
		Context: ctx,
	}, cli.authInfo)
	return err
}

func (cli HarborCli) CreateUser(ctx context.Context, opt *CreateUserOpt) (*User, error) {
	prettyCreateUserOpt(opt)
	_, err := cli.api.User.CreateUser(&huser.CreateUserParams{
		UserReq: &hmodels.UserCreationReq{
			Comment:  "Created by BuGit",
			Email:    opt.Email,
			Password: opt.Password,
			Realname: opt.RealName,
			Username: opt.StudentID,
		},
		Context: ctx,
	}, cli.authInfo)
	if err != nil {
		return nil, err
	}
	return cli.GetUser(ctx, opt.StudentID)
}

func (cli HarborCli) GetProject(ctx context.Context, projectName string) (*Project, error) {
	projectName = PrettyName(projectName)
	respOK, err := cli.api.Project.GetProject(&hproject.GetProjectParams{
		ProjectNameOrID: projectName,
		Context:         ctx,
	}, cli.authInfo)
	if err != nil {
		return nil, err
	}
	resp := respOK.GetPayload()
	if resp == nil {
		return nil, ErrNotFound
	}
	return &Project{Name: resp.Name, IntID: int64(resp.ProjectID)}, nil
}

func (cli HarborCli) CreateProject(ctx context.Context, projectName string) (*Project, error) {
	projectName = PrettyName(projectName)
	_, err := cli.api.Project.CreateProject(&hproject.CreateProjectParams{
		XRequestID:              nil,
		XResourceNameInLocation: nil,
		Project: &hmodels.ProjectReq{
			ProjectName:  projectName,
			Public:       getBoolPtr(true),
			StorageLimit: getInt64Ptr(-1),
		},
		Context: ctx,
	}, cli.authInfo)
	if err != nil {
		return nil, err
	}

	return cli.GetProject(ctx, projectName)
}

func (cli HarborCli) DeleteProject(_ context.Context, _ *Project) error {
	// FIXME 目前通过Harbor API删除project和repo会有bug (https://github.com/goharbor/harbor/issues/15611)
	// FIXME 先假意删除
	return nil
	//for {
	//	respOK, err := cli.api.Repository.ListRepositories(&hrepo.ListRepositoriesParams{
	//		PageSize:    getInt64Ptr(100),
	//		ProjectName: p.Name,
	//		Context:     ctx,
	//	}, cli.authInfo)
	//	if err != nil {
	//		return err
	//	}
	//	resp := respOK.GetPayload()
	//	if len(resp) <= 0 {
	//		break
	//	}
	//
	//	// 并行删除所有的repo
	//	group, ctx := errgroup.WithContext(ctx)
	//	for _, repo := range resp {
	//		group.Go(func() error {
	//			// FIXME 确定一下，这里需不需要对 repoName 进行拆分
	//			return cli.DeleteRepo(ctx, p.Name, repo.Name)
	//		})
	//	}
	//	if err := group.Wait(); err != nil{
	//		return err
	//	}
	//}
	//
	//// 然后删除project
	//_, err := cli.api.Project.DeleteProject(&hproject.DeleteProjectParams{
	//	ProjectNameOrID: strconv.FormatInt(p.IntID, 10),
	//	Context:         ctx,
	//}, cli.authInfo)
	//
	//return err
}

func (cli HarborCli) DeleteRepo(ctx context.Context, projectName, repoName string) error {
	_, err := cli.api.Repository.DeleteRepository(&hrepo.DeleteRepositoryParams{
		ProjectName:    projectName,
		RepositoryName: repoName,
		Context:        ctx,
	}, cli.authInfo)
	return err
}

func (cli HarborCli) ListMember(ctx context.Context, u *User, projectNameOrID string) ([]*hmodels.ProjectMemberEntity, error) {
	respOK, err := cli.api.Member.ListProjectMembers(&hmember.ListProjectMembersParams{
		Entityname:      &(u.Name),
		ProjectNameOrID: projectNameOrID,
		Context:         ctx,
	}, cli.authInfo)
	if err != nil {
		return nil, err
	}
	return respOK.GetPayload(), nil
}

func (cli HarborCli) CheckOwner(ctx context.Context, u *User, projectName string) (bool, error) {
	members, err := cli.ListMember(ctx, u, projectName)
	if err != nil {
		return false, err
	}
	return len(members) > 0, nil
}

func (cli HarborCli) AddOwner(ctx context.Context, u *User, p *Project) error {
	return cli.addMember(ctx, u, p, 2)
}

func (cli HarborCli) addMember(ctx context.Context, u *User, p *Project, roleID int64) error {
	_, err := cli.api.Member.CreateProjectMember(&hmember.CreateProjectMemberParams{
		ProjectMember: &hmodels.ProjectMember{
			MemberUser: &hmodels.UserEntity{
				UserID:   u.IntID,
				Username: u.Name,
			},
			RoleID: roleID,
		},
		ProjectNameOrID: strconv.FormatInt(p.IntID, 10),
		Context:         ctx,
	}, cli.authInfo)
	return err
}

func (cli HarborCli) RemoveMember(ctx context.Context, u *User, p *Project) error {
	pid := strconv.FormatInt(p.IntID, 10)
	resp, err := cli.ListMember(ctx, u, pid)
	if err != nil {
		return err
	}
	if len(resp) <= 0 {
		return ErrNotFound
	}

	_, err = cli.api.Member.DeleteProjectMember(&hmember.DeleteProjectMemberParams{
		Mid:             resp[0].ID,
		ProjectNameOrID: pid,
		Context:         nil,
	}, cli.authInfo)
	return err
}
