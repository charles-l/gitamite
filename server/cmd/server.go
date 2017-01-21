package main

import (
	// web framework
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"

	// API and server functionality
	"github.com/charles-l/gitamite"
	"github.com/charles-l/gitamite/server/context"
	"github.com/charles-l/gitamite/server/helper"
	"github.com/charles-l/gitamite/server/model"
	"github.com/charles-l/gitamite/server/route"

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
	db := model.InitDB()
	defer db.Close()
	log.Printf("loaded DB")

	gitamite.LoadConfig(gitamite.Server)
	repos := make(map[string]*model.Repo)

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
		repos[name] = model.LoadRepository(name, p)
	}

	e := echo.New()
	e.Use(func(h echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cc := &context.Context{c, repos}
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
		"repo_path": func(r *model.Repo) string {
			return route.RepoPath(r)
		},
		"tree_entry_path": func(r *model.Repo, c *model.Commit, t model.TreeEntry) string {
			if t.Type == git.ObjectBlob {
				return path.Join(route.RepoPath(r), "blob", t.DirPath, t.Name)
			} else if t.Type == git.ObjectTree {
				if t.DirPath == "" {
					return "/"
				} else {
					if t.Name == ".." { // TODO: simplify
						return path.Join(route.RepoPath(r), "tree", t.DirPath)
					} else {
						return path.Join(route.RepoPath(r), "tree", t.DirPath, t.Name)
					}
				}
			}
			return ""
		},
		"commit_path": func(r *model.Repo, c *model.Commit) string {
			return route.CommitPath(r, c)
		},
		"user_path": func(u *model.User) string {
			return route.UserPath(u)
		},
		// TODO: make this less garbage
		"blob_path": func(r *model.Repo, b *model.Blob) string {
			return route.BlobPath(r, nil, b)
		},
		"blame_path": func(r *model.Repo, b *model.Blob) string {
			return route.BlamePath(r, b)
		},
		"markdown": func(args ...interface{}) template.HTML {
			// TODO: cache this instead of parsing every time
			s := blackfriday.MarkdownCommon([]byte(fmt.Sprintf("%s", args...)))
			return template.HTML(s)
		},
		"is_file": func(t model.TreeEntry) bool {
			return t.Type == git.ObjectBlob
		},
		"highlight": func(b *model.Blob) template.HTML {
			return model.HighlightedBlobHTML(b)
		},
		// TODO: figure out how to combine these render funcs
		"render_blob": func(b *model.Blob) template.HTML {
			buf := bytes.NewBufferString("<table class=\"diff highlight\">")

			for nu, l := range strings.Split(string(model.HighlightedBlobHTML(b)), "\n") {
				buf.WriteString("<tr><td class=\"lineno\">" + strconv.Itoa(nu+1) + "</td><td>" + l + "</td></tr>")
			}

			buf.WriteString("</table>")
			return template.HTML(buf.String())
		},
		"render_blame": func(b *model.Blame) template.HTML {
			buf := bytes.NewBufferString("<table class=\"diff highlight\">")

			for i, u := range b.Users {
				buf.WriteString("<tr><td><a href=\"" + route.UserPath(u) + "\">" + u.Name + "</a></td><td class=\"lineno\">" + strconv.Itoa(i+1) + "</td><td>" + string(b.Data[i]) + "</td></tr>")
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
		"highlight_blobs": func(blobs []*model.Blob) template.HTML {
			var wg sync.WaitGroup
			outChan := make(chan string, len(blobs))
			for i := range blobs {
				wg.Add(1)
				go func(b *model.Blob) {
					defer wg.Done()
					outChan <- string(model.HighlightedBlobHTML(b))
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
		"render_commit_graph": func(repo *model.Repo) template.HTML {
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
				Repo  *model.Repo
				Error string
			}{
				nil,
				e.Error(),
			})
		} else {
			c.JSON(400, struct{ Error string }{e.Error()})
		}
	}

	route.Setup(e)

	e.Logger.Fatal(e.Start(":8000"))
}
