package handler

import (
	"github.com/charles-l/gitamite"
	"github.com/charles-l/gitamite/server/helper"

	"github.com/labstack/echo"

	"net/http"
)

// TODO: clean this up
func Diff(c echo.Context) error {
	repo, err := helper.Repo(c)
	if err != nil {
		return err
	}

	commitA, err := repo.LookupCommit(c.Param("oidA"))
	if err != nil {
		return err
	}

	var commitB *gitamite.Commit
	if c.Param("oidB") == "" {
		if commitA.ParentCount() > 0 {
			commitB = gitamite.MakeCommit(commitA.Parent(0))
		} else {
			commitB = nil
		}
	} else {
		commitB, err = repo.LookupCommit(c.Param("oidB"))
		if err != nil {
			return err
		}
	}

	diff := gitamite.GetDiff(repo, commitA, commitB)

	c.Render(http.StatusOK, "diff", struct {
		Repo *gitamite.Repo
		Diff *gitamite.Diff
	}{
		repo,
		&diff,
	})
	return nil
}
