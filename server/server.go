package main

import (
	// web framework
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"

	// API and server functionality
	"github.com/charles-l/gitamite"
	"github.com/charles-l/gitamite/server/context"
	"github.com/charles-l/gitamite/server/handler"

	// better templates
	"github.com/unrolled/render"

	// markdown renderer
	"github.com/russross/blackfriday"

	// helper
	"github.com/dustin/go-humanize"

	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"
)

type RenderWrapper struct {
	rnd *render.Render
}

func (r *RenderWrapper) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	err := r.rnd.HTML(w, 0, name, data)
	if err != nil {
		log.Print(err)
	}
	return err
}

func main() {
	gitamite.GlobalConfig = gitamite.ParseConfig(gitamite.ConfigPath)
	if gitamite.GlobalConfig == nil {
		fmt.Fprintf(os.Stderr, "Need config file in %s\n", gitamite.ConfigPath)
		return
	}

	repos := make(map[string]*gitamite.Repo)

	matches, _ := filepath.Glob(path.Join(gitamite.GlobalConfig.RepoDir, "*"))
	for _, p := range matches {
		log.Printf("loading repo from %s\n", p)
		name := filepath.Base(p)
		repos[name] = gitamite.LoadRepository(name, p)
	}

	e := echo.New()
	e.Use(func(h echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cc := &server.Context{c, repos}
			return h(cc)
		}
	})

	e.Pre(middleware.RemoveTrailingSlash())

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
		"path": func(urlables ...gitamite.URLable) string {
			var r []string
			for _, u := range urlables {
				r = append(r, u.URL())
			}
			return path.Join(r...)
		},
		"markdown": func(args ...interface{}) template.HTML {
			// TODO: cache this instead of parsing every time
			s := blackfriday.MarkdownCommon([]byte(fmt.Sprintf("%s", args...)))
			return template.HTML(s)
		},
	}

	r := &RenderWrapper{render.New(render.Options{
		Layout: "layout",
		Funcs:  []template.FuncMap{templateFuncs},
	})}
	e.Renderer = r

	e.GET("/repo/:repo", handler.FileTree)
	e.GET("/repo/:repo/refs", handler.Refs)

	e.GET("/repo/:repo/commits", handler.Commits)
	e.GET("/repo/:repo/:ref/commits", handler.Commits)

	e.GET("/repo/:repo/blob/*", handler.File)
	e.GET("/repo/:repo/commit/:commit/blob/*", handler.File)

	e.GET("/repo/:repo/tree/*", handler.FileTree)
	e.GET("/repo/:repo/commit/:commit/tree/*", handler.FileTree)

	e.GET("/repo/:repo/commit/:oidA", handler.Diff)

	e.POST("/repo", handler.CreateRepo)

	e.Logger.Fatal(e.Start(":8000"))
}
