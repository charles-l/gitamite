package main

import (
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"

	"github.com/dustin/go-humanize"
	"github.com/libgit2/git2go"

	"github.com/charles-l/gitamite"
	"github.com/charles-l/gitamite/server/context"
	"github.com/charles-l/gitamite/server/handler"

	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"time"
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

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func main() {
	repo := loadRepository("gititup", "..")
	defer repo.Free()

	e := echo.New()
	e.Use(func(h echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cc := &server.Context{c, repo}
			return h(cc)
		}
	})

	e.Pre(middleware.AddTrailingSlash())

	templateFuncs := template.FuncMap{
		"humanizeTime": func(t time.Time) string {
			return humanize.Time(t)
		},
		"s_ify": func(str string, n int) string {
			if n == 1 {
				return fmt.Sprintf("%d %s", n, str)
			} else {
				return fmt.Sprintf("%d %ss", n, str)
			}
		},
	}

	t := &Template{
		templates: template.Must(template.New("t").Funcs(templateFuncs).ParseGlob("templates/*")),
	}
	e.Renderer = t

	e.POST("/repo", handler.CreateRepo)

	e.GET("/repo/:repo/", handler.CommitsHandler)
	e.GET("/repo/:repo/commits/", handler.CommitsHandler)
	e.GET("/repo/:repo/commit/:oidA", handler.DiffHandler)
	e.GET("/repo/:repo/blob/*", handler.FileHandler)
	e.GET("/repo/:repo/tree/*", handler.FileTreeHandler)

	e.Logger.Fatal(e.Start(":8000"))
}
