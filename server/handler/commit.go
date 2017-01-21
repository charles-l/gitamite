package handler

import (
	"net/http"

	"github.com/charles-l/gitamite/server/helper"
	"github.com/charles-l/gitamite/server/model"
	"github.com/labstack/echo"
)

func FullCommits(c echo.Context) error {
	repo, err := helper.RepoParam(c)
	if err != nil {
		return err
	}

	log := repo.CommitLog(nil)

	c.Render(http.StatusOK, "log",
		struct {
			Repo    *model.Repo
			Commits []*model.Commit
		}{
			repo,
			log,
		})
	return nil
}

func Commits(c echo.Context) error {
	repo, err := helper.RepoParam(c)
	if err != nil {
		return err
	}

	ref, err := helper.RefParam(c, true)
	if err != nil {
		return err
	}

	var log []*model.Commit
	if ref == nil { // i'm just being explicit
		log = repo.CommitLog(nil)
	} else {
		log = repo.CommitLog(ref)
	}

	c.Render(http.StatusOK, "log",
		struct {
			Repo    *model.Repo
			Commits []*model.Commit
		}{
			repo,
			log,
		})
	return nil
}
