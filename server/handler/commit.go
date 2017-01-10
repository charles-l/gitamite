package handler

import (
	"net/http"

	"github.com/charles-l/gitamite"
	"github.com/charles-l/gitamite/server/helper"
	"github.com/labstack/echo"
)

func Commits(c echo.Context) error {
	repo, err := helper.Repo(c)
	if err != nil {
		return err
	}

	ref, err := helper.Ref(c)
	if err != nil {
		// TODO: pass server name
		c.Render(http.StatusOK, "empty", struct {
			Repo *gitamite.Repo
		}{
			repo,
		})
		return nil
	}

	log := gitamite.GetCommitLog(repo, ref)

	c.Render(http.StatusOK, "log",
		struct {
			Repo    *gitamite.Repo
			Commits []gitamite.Commit
		}{
			repo,
			log,
		})
	return nil
}
