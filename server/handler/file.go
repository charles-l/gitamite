package handler

import (
	"github.com/charles-l/gitamite/server/helper"
	"github.com/charles-l/gitamite/server/model"

	"github.com/labstack/echo"

	"fmt"
	"log"
	"net/http"
	"path"
)

func File(c echo.Context) error {
	repo, err := helper.RepoParam(c)
	if err != nil {
		return err
	}

	commit, err := helper.CommitParam(c)
	if err != nil {
		return err
	}

	s, err := repo.ReadBlob(commit, helper.PathParam(c))
	if err != nil {
		return err
	}

	ext := path.Ext(helper.PathParam(c))
	if ext != "" {
		ext = ext[1:]
	} else {
		ext = "text"
	}

	c.Render(http.StatusOK, "file", struct {
		Repo *model.Repo
		Blob *model.Blob
	}{
		repo,
		s,
	})
	return nil
}

func Blame(c echo.Context) error {
	// TODO: figure out how to pull the repo check out further so it's not duplicated everywhere
	repo, err := helper.RepoParam(c)
	if err != nil {
		return err
	}

	commit, err := helper.CommitParam(c)
	if err != nil {
		return err
	}

	s, err := repo.ReadBlobBlame(commit, helper.PathParam(c))
	if err != nil {
		log.Printf("read blob: %s", err)
		return fmt.Errorf("failed to get blob")
	}

	c.Render(http.StatusOK, "blame", struct {
		Repo  *model.Repo
		Blame *model.Blame
	}{
		repo,
		s,
	})
	return nil
}
