package handler

import (
	"net/http"

	"github.com/charles-l/gitamite"
	"github.com/charles-l/gitamite/server/helper"
	"github.com/labstack/echo"
)

func Commits(c echo.Context) error {
	repo, _ := helper.Repo(c)

	ref, err := helper.Ref(c)
	if err != nil {
		return err
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
