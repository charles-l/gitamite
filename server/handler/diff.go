package handler

import (
	"github.com/charles-l/gitamite/server/helper"
	"github.com/charles-l/gitamite/server/model"

	"github.com/labstack/echo"

	"net/http"
)

// TODO: clean this up
func Diff(c echo.Context) error {
	repo, err := helper.RepoParam(c)
	if err != nil {
		return err
	}

	commitA, err := repo.LookupCommit(c.Param("oidA"))
	if err != nil {
		return err
	}

	var commitB *model.Commit
	if c.Param("oidB") == "" {
		if commitA.ParentCount() > 0 {
			commitB = model.MakeCommit(commitA.Parent(0))
		} else {
			commitB = nil
		}
	} else {
		commitB, err = repo.LookupCommit(c.Param("oidB"))
		if err != nil {
			return err
		}
	}

	diff := model.GetDiff(repo, commitA, commitB)

	c.Render(http.StatusOK, "diff", struct {
		Repo *model.Repo
		Diff *model.Diff
	}{
		repo,
		&diff,
	})
	return nil
}
