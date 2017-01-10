package handler

import (
	"github.com/charles-l/gitamite"
	"github.com/charles-l/gitamite/server/helper"

	"fmt"
	"github.com/labstack/echo"
	"log"
	"net/http"
)

func FileTree(c echo.Context) error {
	path := "/"
	repo, err := helper.Repo(c)
	if err != nil {
		return err
	}

	commit, err := helper.Commit(c)
	if err != nil {
		// TODO: pass server name
		c.Render(http.StatusOK, "empty", struct {
			Repo *gitamite.Repo
		}{
			repo,
		})
		return nil
	}

	path = helper.PathParam(c)

	t, _ := commit.Tree()

	readme := ""
	if buf, err := repo.ReadBlob(commit, "README.md"); err == nil {
		readme = string(buf)
	}

	var entries []gitamite.TreeEntry
	if path == "/" || path == "" {
		entries = gitamite.GetTreeEntries(t, "/")
	} else {
		entries, err = gitamite.GetSubTree(t, path)
		if err != nil {
			log.Printf("Filetree error: %v", err)
			return fmt.Errorf("Failed to get file tree")
		}
	}

	c.Render(http.StatusOK, "filelist",
		struct {
			Repo    *gitamite.Repo
			Entries []gitamite.TreeEntry
			README  string
		}{
			repo,
			entries,
			readme,
		})
	return nil
}
