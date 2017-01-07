package handler

import (
	"net/http"

	"fmt"

	"github.com/charles-l/gitamite"
	"github.com/gorilla/mux"
)

func parseRef(r *gitamite.Repo, vars map[string]string) (gitamite.Ref, error) {
	refstr, exist := vars["ref"]
	if !exist {
		refstr = "master"
	}
	ref, err := r.LookupRef(refstr)
	return ref, err
}

func CommitsHandler(c *Context, w http.ResponseWriter, r *http.Request) (int, error) {
	ref, err := parseRef(c.Repo, mux.Vars(r))
	if err != nil {
		return 404, fmt.Errorf("ref not found")
	}

	log := gitamite.GetCommitLog(c.Repo, ref)

	c.Render(w, "log",
		struct {
			Repo    *gitamite.Repo
			Commits []gitamite.Commit
		}{
			c.Repo,
			log,
		})
	return 200, nil
}
