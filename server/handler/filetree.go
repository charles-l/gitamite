package handler

import (
	"github.com/charles-l/gitamite"
	"github.com/gorilla/mux"
	"net/http"
)

func FileTreeHandler(c *Context, w http.ResponseWriter, r *http.Request) (int, error) {
	vars := mux.Vars(r)
	path := "/"
	var commit gitamite.Commit
	if vars["path"] != "" {
		path = vars["path"]
	}
	if vars["commit"] != "" {
		commit, _ = c.Repo.LookupCommit(vars["commit"])
	} else {
		commit, _ = c.Repo.DefaultCommit()
	}
	c.Render.RenderFileTree(w, c.Repo, &commit, path)
	return 200, nil
}
