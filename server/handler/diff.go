package handler

import (
	"github.com/charles-l/gitamite"
	"github.com/charles-l/gitamite/server/context"
	"github.com/labstack/echo"

	"net/http"
)

// TODO: clean this up
func Diff(c echo.Context) error {
	t, err := c.(*server.Context).Repo().LookupCommit(c.Param("oidA"))
	if err != nil {
		return err
	}
	commitA := &t

	var commitB *gitamite.Commit
	if c.Param("oidB") == "" {
		if commitA.ParentCount() > 0 {
			commitB = &gitamite.Commit{commitA.Parent(0)}
		} else {
			commitB = nil
		}
	} else {
		t, err = c.(*server.Context).Repo().LookupCommit(c.Param("oidB"))
		if err != nil {
			return err
		}
		commitB = &t
	}

	diff := gitamite.GetDiff(c.(*server.Context).Repo(), commitA, commitB)

	c.Render(http.StatusOK, "diff", struct {
		Repo *gitamite.Repo
		Diff gitamite.Diff
	}{
		c.(*server.Context).Repo(),
		diff,
	})
	return nil
}
