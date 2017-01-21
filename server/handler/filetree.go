package handler

import (
	"github.com/charles-l/gitamite/server/helper"
	"github.com/charles-l/gitamite/server/model"

	"fmt"
	"github.com/labstack/echo"
	"log"
	"net/http"
)

func FileTree(c echo.Context) error {
	path := "/"
	repo, err := helper.RepoParam(c)
	if err != nil {
		return err
	}

	commit, err := helper.CommitParam(c)
	if err != nil {
		// TODO: pass server name
		c.Render(http.StatusOK, "empty", struct {
			Repo *model.Repo
		}{
			repo,
		})
		return nil
	}

	path = helper.PathParam(c)

	t, _ := commit.Tree()

	readme := ""
	if blob, err := repo.ReadBlob(commit, "README.md"); err == nil {
		readme = string(blob.ByteArray())
	}

	var entries []model.TreeEntry
	if path == "/" || path == "" {
		entries = model.GetTreeEntries(t, "/")
	} else {
		entries, err = model.GetSubTree(t, path)
		if err != nil {
			log.Printf("Filetree error: %v", err)
			return fmt.Errorf("Failed to get file tree")
		}
	}

	c.Render(http.StatusOK, "filelist",
		struct {
			Repo    *model.Repo
			Entries []model.TreeEntry
			README  string
		}{
			repo,
			entries,
			readme,
		})
	return nil
}
