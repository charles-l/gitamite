package handler

import (
	"github.com/charles-l/gitamite"
	"github.com/charles-l/gitamite/server/helper"

	"github.com/charles-l/pygments"
	"github.com/labstack/echo"

	"html"
	"html/template"

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

	o, err := pygments.Highlight(s, ext, "html", "utf-8")
	if err != nil {
		o = "<pre>" + html.EscapeString(string(s)) + "</pre>"
	}
	c.Render(http.StatusOK, "file", struct {
		Repo     *gitamite.Repo
		Text     template.HTML
		BlameURL string
	}{
		repo,
		template.HTML(o),
		// TODO: pass the entity instead and use URLable
		path.Join(repo.URL(), "blame", "blob", helper.PathParam(c)),
	})
	return nil
}

func FileBlame(c echo.Context) error {
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
		Repo *gitamite.Repo
		Text string

		BlameURL string
		// TODO: pass the entity instead and use URLable
		FileURL string
	}{
		repo,
		string(s),
		"",
		path.Join(repo.URL(), "blob", helper.PathParam(c)),
	})
	return nil
}
