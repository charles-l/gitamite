package handler

import (
	"github.com/charles-l/gitamite"
	"github.com/charles-l/gitamite/server/context"
	"github.com/labstack/echo"

	"net/http"
)

func FileHandler(c echo.Context) error {
	var commit gitamite.Commit
	commitstr := c.Param("commit")
	if commitstr == "" {
		commit, _ = c.(*server.Context).Repo.DefaultCommit()
	} else {
		var err error
		commit, err = c.(*server.Context).Repo.LookupCommit(commitstr)
		if err != nil {
			return err
		}
	}

	s, err := c.(*server.Context).Repo.ReadBlob(&commit, c.Param("path"))
	if err != nil {
		return err
	}

	c.Render(http.StatusOK, "file", struct {
		Repo *gitamite.Repo
		Text string
	}{
		&c.(*server.Context).Repo,
		string(s),
	})
	return nil
}
