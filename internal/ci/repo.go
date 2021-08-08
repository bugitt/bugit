package ci

import (
	"os"
	"path"
	"path/filepath"

	"git.scs.buaa.edu.cn/iobs/bugit/internal/db"
	"git.scs.buaa.edu.cn/iobs/bugit/internal/tool"
	"github.com/artdarek/go-unzip"
	"github.com/bugitt/git-module"
	"github.com/unknwon/com"
)

func loadRepo(ctx *Context) (err error) {
	err = ctx.updateStage(db.LoadRepoStart, -1)
	if err != nil {
		return
	}

	// 如果已经存在了，那么就不用再load一次了
	if com.IsDir(ctx.path) {
		return nil
	}

	gitRepo, err := git.Open(ctx.repo.RepoPath())
	if err != nil {
		return
	}
	gitCommit, err := gitRepo.CatFileCommit(ctx.commit)
	if err != nil {
		return
	}
	hash := tool.ShortSHA1(ctx.commit)
	archivePath := filepath.Join(gitRepo.Path(), "archives", "zip")
	if !com.IsDir(archivePath) {
		if err := os.MkdirAll(archivePath, os.ModePerm); err != nil {
			return err
		}
	}
	archivePath = path.Join(archivePath, hash+".zip")
	if !com.IsFile(archivePath) {
		if err := gitCommit.CreateArchive(git.ArchiveZip, archivePath); err != nil {
			return err
		}
	}

	if !com.IsDir(ctx.path) {
		uz := unzip.New(archivePath, ctx.path)
		if err := uz.Extract(); err != nil {
			return err
		}
	}

	return ctx.updateStage(db.LoadRepoEnd, -1)
}
