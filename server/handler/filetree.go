package handler

import (
	"github.com/charles-l/gitamite"
	"github.com/charles-l/gitamite/server/context"
	"github.com/labstack/echo"
	"net/http"
)

func FileTree(c echo.Context) error {
	path := "/"
	repo := c.(*server.Context).Repo()
	var commit gitamite.Commit
	if c.Param("*") != "" {
		path = c.Param("*")
	}
	if c.Param("commit") != "" {
		commit, _ = repo.LookupCommit(c.Param("commit"))
	} else {
		commit, _ = repo.DefaultCommit()
	}

	////////////
	t, _ := commit.Tree()

	readme := ""
	if buf, err := repo.ReadBlob(&commit, "README.md"); err == nil {
		readme = string(buf)
	}

	var entries []gitamite.TreeEntry
	if path == "/" || path == "" {
		entries = gitamite.GetTreeEntries(t, "/")
	} else {
		entries = gitamite.GetSubTree(t, path)
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
