package handler

import (
	"github.com/charles-l/gitamite"
	"github.com/charles-l/gitamite/server/context"
	"github.com/labstack/echo"

	"net/http"
)

// TODO: clean this up
func DiffHandler(c echo.Context) error {
	commitA, err := c.(*server.Context).Repo.LookupCommit(c.Param("oidA"))
	if err != nil {
		return err
	}
	defer commitA.Free()

	var commitB gitamite.Commit
	if c.Param("oidB") == "" {
		commitB = gitamite.Commit{commitA.Parent(0)}
	} else {
		commitB, err = c.(*server.Context).Repo.LookupCommit(c.Param("oidB"))
		if err != nil {
			return err
		}
	}

	diff := gitamite.GetDiff(&c.(*server.Context).Repo, &commitA, &commitB)

	c.Render(http.StatusOK, "diff", struct {
		Repo *gitamite.Repo
		Diff gitamite.Diff
	}{
		&c.(*server.Context).Repo,
		diff,
	})
	return nil
}
