package handler

import (
	"github.com/charles-l/gitamite"

	"github.com/gorilla/mux"

	"fmt"
	"net/http"
)

// TODO: clean this up
func DiffHandler(c *Context, w http.ResponseWriter, r *http.Request) (int, error) {
	vars := mux.Vars(r)

	commitA, err := c.Repo.LookupCommit(vars["oidA"])
	if err != nil {
		return 404, fmt.Errorf("unable to find commit: " + vars["oidA"])
	}
	defer commitA.Free()

	var commitB gitamite.Commit
	if vars["oidB"] == "" {
		commitB = gitamite.Commit{commitA.Parent(0)}
	} else {
		commitB, err = c.Repo.LookupCommit(vars["oidB"])
		if err != nil {
			return 404, fmt.Errorf("unable to find commit: " + vars["oidB"])
		}
	}

	diff := gitamite.GetDiff(c.Repo, &commitA, &commitB)

	c.Render(w, "diff", struct {
		Repo *gitamite.Repo
		Diff gitamite.Diff
	}{
		c.Repo,
		diff,
	})
	return 200, nil
}
