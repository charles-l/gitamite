package handler

import (
	"github.com/charles-l/gitamite"
	"github.com/charles-l/gitamite/server/helper"
	"net/http"
)

type Context struct {
	Render helper.Renderer
	Repo   *gitamite.Repo
}

type HandleFunc func(*Context, http.ResponseWriter, *http.Request) (int, error)

type AppHandler struct {
	*Context
	H HandleFunc
}

func (h AppHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	status, err := h.H(h.Context, w, r)
	if err != nil {
		// TODO: handle other error codes here
		// TODO: make pretty error page
		http.Error(w, err.Error(), status)
	}
}
