package handler

import (
	"github.com/charles-l/gitamite"
	"github.com/charles-l/gitamite/server/helper"

	"github.com/labstack/echo"

	"net/http"
)

func File(c echo.Context) error {
	repo, _ := helper.Repo(c)
	commit, err := helper.Commit(c)
	if err != nil {
		return err
	}

	s, err := repo.ReadBlob(commit, c.Param("*"))
	if err != nil {
		return err
	}

	c.Render(http.StatusOK, "file", struct {
		Repo *gitamite.Repo
		Text string
	}{
		repo,
		string(s),
	})
	return nil
}
