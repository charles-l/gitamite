package main

import (
	// web framework
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"

	// API and server functionality
	"github.com/charles-l/gitamite"
	"github.com/charles-l/gitamite/server/context"
	"github.com/charles-l/gitamite/server/handler"
	"github.com/charles-l/gitamite/server/helper"

	// better templates
	"github.com/unrolled/render"

	// markdown renderer
	"github.com/russross/blackfriday"

	"github.com/dustin/go-humanize"
	"github.com/libgit2/git2go"

	"bytes"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
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
	db := gitamite.InitDB()
	defer db.Close()
	log.Printf("loaded DB")

	gitamite.LoadConfig(gitamite.Server)
	repos := make(map[string]*gitamite.Repo)

	repoDir, err := gitamite.GetConfigValue("repo_dir")
	if err != nil {
		// bail out if no repo path has been set
		// we really don't want to accidentally overwrite stuff in /
		return
	}
	matches, _ := filepath.Glob(path.Join(repoDir, "*"))
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
		"is_file": func(t gitamite.TreeEntry) bool {
			return t.Type == git.ObjectBlob
		},
		"highlight": func(b *gitamite.Blob) template.HTML {
			return gitamite.HighlightedBlobHTML(b)
		},
		// TODO: figure out how to combine these render funcs
		"render_blob": func(b *gitamite.Blob) template.HTML {
			buf := bytes.NewBufferString("<table class=\"diff highlight\">")

			for nu, l := range strings.Split(string(gitamite.HighlightedBlobHTML(b)), "\n") {
				buf.WriteString("<tr><td class=\"lineno\">" + strconv.Itoa(nu+1) + "</td><td>" + l + "</td></tr>")
			}

			buf.WriteString("</table>")
			return template.HTML(buf.String())
		},
		"render_blame": func(b *gitamite.Blame) template.HTML {
			buf := bytes.NewBufferString("<table class=\"diff highlight\">")

			for i, u := range b.Users {
				buf.WriteString("<tr><td><a href=\"" + u.URL() + "\">" + u.Name + "</a></td><td class=\"lineno\">" + strconv.Itoa(i+1) + "</td><td>" + string(b.Data[i]) + "</td></tr>")
			}

			buf.WriteString("</table>")
			return template.HTML(buf.String())
		},
		"diff_add": func(l git.DiffLine) bool {
			return l.Origin == git.DiffLineAddition
		},
		"diff_del": func(l git.DiffLine) bool {
			return l.Origin == git.DiffLineDeletion
		},
		"highlight_blobs": func(blobs []*gitamite.Blob) template.HTML {
			var wg sync.WaitGroup
			outChan := make(chan string, len(blobs))
			for i := range blobs {
				wg.Add(1)
				go func(b *gitamite.Blob) {
					defer wg.Done()
					outChan <- string(gitamite.HighlightedBlobHTML(b))
				}(blobs[i])
			}
			wg.Wait()
			a := make([]string, len(blobs))
			for i := range blobs {
				a[i] = <-outChan
			}
			return template.HTML(strings.Join(a, ""))
		},
		"eqv": func(a interface{}, b interface{}) bool {
			return a == b
		},
		"render_commit_graph": func(repo *gitamite.Repo) template.HTML {
			return template.HTML(helper.RenderLogTree(repo))
		},
	}

	r := &RenderWrapper{render.New(render.Options{
		Layout: "layout",
		Funcs:  []template.FuncMap{templateFuncs},
	})}

	e.Renderer = r
	e.HTTPErrorHandler = func(e error, c echo.Context) {
		if c.Request().Header.Get("Content-Type") != "application/json" {
			// TODO: don't always blame teh user :P
			c.Render(http.StatusBadRequest, "error", struct {
				Repo  *gitamite.Repo
				Error string
			}{
				nil,
				e.Error(),
			})
		} else {
			c.JSON(400, struct{ Error string }{e.Error()})
		}
	}

	e.Static("/a", "pub")

	e.GET("/", handler.Repos)

	e.GET("/repo/:repo", handler.FileTree)
	e.GET("/repo/:repo/refs", handler.Refs)

	e.GET("/repo/:repo/commits", handler.FullCommits)
	e.GET("/repo/:repo/:ref/commits", handler.Commits)

	e.GET("/repo/:repo/blob/*", handler.File)
	e.GET("/repo/:repo/blame/*", handler.Blame)
	e.GET("/repo/:repo/commit/:commit/blob/*", handler.File)

	e.GET("/repo/:repo/tree/*", handler.FileTree)
	e.GET("/repo/:repo/commit/:commit/tree/*", handler.FileTree)

	e.GET("/repo/:repo/commit/:oidA", handler.Diff)

	e.POST("/repo", handler.CreateRepo)
	e.DELETE("/repo", handler.DeleteRepo)

	e.GET("/user/:email", handler.User)

	e.Logger.Fatal(e.Start(":8000"))
}
