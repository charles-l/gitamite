package handler

import (
	"github.com/charles-l/gitamite"
	"github.com/charles-l/gitamite/server/helper"

	"github.com/labstack/echo"

	"fmt"
	"log"
	"net/http"
	"path"
)

func File(c echo.Context) error {
	repo, err := helper.Repo(c)
	if err != nil {
		return err
	}

	commit, err := helper.Commit(c)
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
		Repo *gitamite.Repo
		Blob *gitamite.Blob

		FileExt  string
		BlameURL string
	}{
		repo,
		s,

		ext,

		// TODO: pass the entity instead and use URLable
		path.Join(repo.URL(), "blame", "blob", helper.PathParam(c)),
	})
	return nil
}

func Blame(c echo.Context) error {
	// TODO: figure out how to pull the repo check out further so it's not duplicated everywhere
	repo, err := helper.Repo(c)
	if err != nil {
		return err
	}

	commit, err := helper.Commit(c)
	if err != nil {
		return err
	}

	s, err := repo.ReadBlobBlame(commit, helper.PathParam(c))
	if err != nil {
		log.Printf("read blob: %s", err)
		return fmt.Errorf("failed to get blob")
	}

	c.Render(http.StatusOK, "file", struct {
		Repo  *gitamite.Repo
		Blame *gitamite.Blame

		BlameURL string
		// TODO: pass the entity instead and use URLable
		FileURL string
	}{
		repo,
		s,

		"",
		path.Join(repo.URL(), "blob", helper.PathParam(c)),
	})
	return nil
}
