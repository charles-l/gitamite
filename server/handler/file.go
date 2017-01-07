package handler

import (
	"fmt"

	"github.com/charles-l/gitamite"
	"github.com/gorilla/mux"

	"net/http"
)

func FileHandler(c *Context, w http.ResponseWriter, r *http.Request) (int, error) {
	vars := mux.Vars(r)
	var commit gitamite.Commit
	commitstr, exists := vars["commit"]
	if !exists {
		commit, _ = c.Repo.DefaultCommit()
	} else {
		var err error
		commit, err = c.Repo.LookupCommit(commitstr)
		if err != nil {
			return 404, fmt.Errorf("unable to find commit")
		}
	}

	s, err := c.Repo.ReadBlob(&commit, vars["path"])
	if err != nil {
		return 404, fmt.Errorf("file not found")
	}

	c.Render(w, "file", struct {
		Repo *gitamite.Repo
		Text string
	}{
		c.Repo,
		string(s),
	})
	return 200, nil
}
