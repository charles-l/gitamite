package main

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/libgit2/git2go"

	"github.com/charles-l/gitamite"
	"github.com/charles-l/gitamite/server/handler"
	"github.com/charles-l/gitamite/server/helper"

	"io/ioutil"
	"log"
)

func loadRepository(name string, path string) gitamite.Repo {
	repo, err := git.OpenRepository(path)
	if err != nil {
		log.Fatal("failed to open repo ", path, ":", err)
	}
	desc, err := ioutil.ReadFile(path + "/.git/description")
	if err != nil {
		log.Print("failed to get repo description ", path, ":", err)
		desc = []byte("")
	}
	return gitamite.Repo{
		name,
		path,
		string(desc),
		repo,
	}
}

func main() {
	repo := loadRepository("gititup", "..")
	defer repo.Free()

	render := helper.CreatePageRenderer()

	context := handler.Context{
		render,
		&repo,
	}

	r := mux.NewRouter().StrictSlash(true)

	repoRouter := r.PathPrefix("/repos/{repo}").Methods("GET").Subrouter()
	repoRouter.Handle("/", handler.AppHandler{&context, handler.CommitsHandler})
	repoRouter.Handle("/commits/", handler.AppHandler{&context, handler.CommitsHandler})
	repoRouter.Handle("/commit/{oidA}/", handler.AppHandler{&context, handler.DiffHandler})
	repoRouter.Handle("/blob/{path:.*}", handler.AppHandler{&context, handler.FileHandler})
	repoRouter.Handle("/tree/{path:.*}/", handler.AppHandler{&context, handler.FileTreeHandler})

	r.Methods("POST").Subrouter().Handle("/repos", handler.CreateHandler{render})

	http.Handle("/", r)
	http.ListenAndServe(":8000", nil)
}
