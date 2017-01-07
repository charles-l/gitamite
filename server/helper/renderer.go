package helper

import (
	"fmt"
	"github.com/charles-l/gitamite"
	"github.com/dustin/go-humanize"
	"html/template"
	"io"
	"log"
	"path/filepath"
	"strings"
	"time"
)

type Renderer func(w io.Writer, name string, i interface{})

func CreatePageRenderer() Renderer {
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
		"path": func(p gitamite.Pathable) string {
			return p.Path()
		},
	}

	templates := make(map[string]*template.Template)
	baseTemp := template.Must(template.ParseGlob("layouts/*")).Funcs(templateFuncs)

	matches, _ := filepath.Glob("templates/*")
	for _, f := range matches {
		basename := filepath.Base(f)
		ext := filepath.Ext(basename)
		templates[strings.TrimSuffix(basename, ext)] = template.Must(template.Must(baseTemp.Clone()).ParseGlob(f))
	}

	return func(w io.Writer, name string, i interface{}) {
		if templates[name] == nil {
			log.Fatal("no such template ", name)
		}
		err := templates[name].ExecuteTemplate(w, "main", i)
		if err != nil {
			log.Fatal("error executing template: ", err)
		}
	}
}

// TODO: remove - i don't like it
func (r Renderer) RenderFileTree(w io.Writer, repo *gitamite.Repo, commit *gitamite.Commit, path string) {
	t, _ := commit.Tree()

	readme := ""
	if buf, err := repo.ReadBlob(commit, "README.md"); err == nil {
		readme = string(buf)
	}

	var entries []gitamite.TreeEntry
	if path == "/" || path == "" {
		entries = gitamite.GetTreeEntries(t, "/")
	} else {
		entries = gitamite.GetSubTree(t, path)
	}

	r(w, "filelist",
		struct {
			Repo    *gitamite.Repo
			Entries []gitamite.TreeEntry
			README  string
		}{
			repo,
			entries,
			readme,
		})
}
