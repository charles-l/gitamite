package handler

import (
	"net/http"

	"github.com/charles-l/gitamite"
	"github.com/charles-l/gitamite/server/helper"
	"github.com/labstack/echo"
)

func FullCommits(c echo.Context) error {
	repo, err := helper.Repo(c)
	if err != nil {
		return err
	}

	log := gitamite.GetCommitLog(repo, nil)

	c.Render(http.StatusOK, "log",
		struct {
			Repo    *gitamite.Repo
			Commits []*gitamite.Commit
		}{
			repo,
			log,
		})
	return nil
}

func Commits(c echo.Context) error {
	repo, err := helper.Repo(c)
	if err != nil {
		return err
	}

	ref, err := helper.Ref(c, true)
	if err != nil {
		return err
	}

	// i know this if statement isn't needed
	// i just wanted it to b clear what's going on here
	var log []*gitamite.Commit
	if ref == nil {
		log = gitamite.GetCommitLog(repo, nil)
	} else {
		log = gitamite.GetCommitLog(repo, ref)
	}

	c.Render(http.StatusOK, "log",
		struct {
			Repo    *gitamite.Repo
			Commits []*gitamite.Commit
		}{
			repo,
			log,
		})
	return nil
}
