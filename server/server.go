package main

import (
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/gorilla/mux"
	"github.com/libgit2/git2go"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type TreeEntry struct {
	// extend with a pointer to the parent tree
	DirPath string

	*git.TreeEntry
}
type Pathable interface {
	Path() string
}

type Repo struct {
	Name        string
	Path        string
	Description string
	_Repo       *git.Repository
}

type Commit struct {
	_Commit *git.Commit
	Message string
	Author  string
	Hash    string
	Date    time.Time
}

type Diff struct {
	_CommitA    *git.Commit
	_CommitB    *git.Commit
	CommitAHash string
	CommitBHash string
	Stats       string
	Patches     []string
}

func (c Commit) Path() string {
	return path.Join("commit", c.Hash)
}

func (t TreeEntry) Path() string {
	if t.Type == git.ObjectBlob {
		return filepath.Join("/", "blob", t.DirPath, t.Name)
	} else if t.Type == git.ObjectTree {
		if t.DirPath == "" {
			return "/"
		} else {
			if t.Name == ".." { // TODO: simplify
				return filepath.Join("/", "tree", t.DirPath)
			} else {
				return filepath.Join("/", "tree", t.DirPath, t.Name)
			}
		}
	}
	log.Fatal("unknown type tree entry type ", t.Type)
	return ""
}

// TODO: get commit log for an arbitrary branch
func getCommitLog(repo *git.Repository, ref *git.Reference) []Commit {
	r, err := repo.Walk()
	if err != nil {
		log.Print("failed to walk repo: ", err)
	}

	r.Push(ref.Target())
	r.Sorting(git.SortTime)
	r.SimplifyFirstParent()

	id := &(git.Oid{})

	var commits []Commit
	for r.Next(id) == nil {
		g, _ := repo.LookupCommit(id)
		c := Commit{g, g.Message(), g.Committer().Name, g.Id().String(), g.Committer().When}
		commits = append(commits, c)
	}
	return commits
}

func getCommitDiff(repo *git.Repository, commit *git.Commit) Diff {
	// TODO check for multiple parent commits
	p, _ := commit.Parent(0).Tree()
	c, _ := commit.Tree()
	o, _ := git.DefaultDiffOptions()
	diff, _ := repo.DiffTreeToTree(p, c, &o)
	defer diff.Free()

	stats, _ := diff.Stats()
	statsStr, _ := stats.String(git.DiffStatsFull, 80)

	r := Diff{
		_CommitA:    commit.Parent(0),
		_CommitB:    commit,
		CommitAHash: p.Id().String(),
		CommitBHash: c.Id().String(),
		Stats:       statsStr,
	}
	n, _ := diff.NumDeltas()
	for i := 0; i < n; i++ {
		patch, _ := diff.Patch(i)

		s, _ := patch.String()
		r.Patches = append(r.Patches, s)

		patch.Free()
	}
	return r
}

func readBlob(repo *git.Repository, commit *git.Commit, filepath string) (string, error) {
	t, err := commit.Tree()
	if err != nil {
		log.Print("invalid tree: ", err)
	}

	te, _ := t.EntryByPath(filepath)
	if te == nil {
		log.Print("no file: ", filepath)
		return "", fmt.Errorf("no such file/blob/tree entry %g", filepath)
	}

	f, err := repo.Lookup(te.Id)
	if err != nil {
		log.Print("invalid file: ", err)
	}

	b, err := f.AsBlob()
	if err != nil {
		log.Print("invalid blob: ", err)
	}

	return string(b.Contents()), nil
}

type Renderer func(w io.Writer, name string, i interface{})

func createPageRenderer() Renderer {
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
		"path": func(p Pathable) string {
			return p.Path()
		},
	}

	templates := make(map[string]*template.Template)
	baseTemp := template.Must(template.ParseGlob("l/*")).Funcs(templateFuncs)

	matches, _ := filepath.Glob("t/*")
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

func loadRepository(name string, path string) Repo {
	repo, err := git.OpenRepository(path)
	if err != nil {
		log.Fatal("failed to open repo ", path, ":", err)
	}
	desc, err := ioutil.ReadFile(path + "/.git/description")
	if err != nil {
		log.Print("failed to get repo description ", path, ":", err)
		desc = []byte("")
	}
	return Repo{
		Name:        name,
		Path:        path,
		Description: string(desc),
		_Repo:       repo,
	}
}

func getParentDir(path string) TreeEntry {
	dir, _ := filepath.Split(path)

	g := git.TreeEntry{
		Name: "..",
		Type: git.ObjectTree,
	}

	t := TreeEntry{dir, &g}
	return t
}

func getTreeEntries(t *git.Tree, treePath string) []TreeEntry {
	var r []TreeEntry
	for i := uint64(0); i < t.EntryCount(); i++ {
		r = append(r, TreeEntry{treePath, t.EntryByIndex(i)})
	}
	return r
}

// TODO: combine getTreeEntry and getSubTree into one function
func getSubTree(t *git.Tree, treePath string) []TreeEntry {
	subentry, _ := t.EntryByPath(treePath)
	if subentry.Type != git.ObjectTree {
		log.Fatal("path is not a subtree ", treePath, " - is ", subentry.Type)
	}
	subtree, _ := t.Object.Owner().LookupTree(subentry.Id)
	return append([]TreeEntry{getParentDir(treePath)}, getTreeEntries(subtree, treePath)...)
}

func getTreeEntry(t *git.Tree, treePath string) TreeEntry {
	e, _ := t.EntryByPath(treePath)
	return TreeEntry{treePath, e}
}

func (r Renderer) renderFileTree(w io.Writer, repo Repo, commit *git.Commit, path string) {
	t, _ := commit.Tree()

	readme := ""
	if buf, err := readBlob(repo._Repo, commit, "README.md"); err == nil {
		readme = string(buf)
	}

	var entries []TreeEntry
	if path == "/" || path == "" {
		entries = getTreeEntries(t, "/")
	} else {
		entries = getSubTree(t, path)
	}

	r(w, "filelist",
		struct {
			Repo    Repo
			Entries []TreeEntry
			README  string
		}{
			repo,
			entries,
			readme,
		})
}

func main() {
	repo := loadRepository("gititup", "..")
	defer repo._Repo.Free()

	render := createPageRenderer()
	master, _ := repo._Repo.LookupBranch("master", git.BranchAll)
	firstCommitObj, _ := master.Reference.Peel(git.ObjectCommit)
	firstCommit, _ := firstCommitObj.AsCommit()

	r := mux.NewRouter()
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		render.renderFileTree(w, repo, firstCommit, "/")
	})

	r.HandleFunc("/log/", func(w http.ResponseWriter, r *http.Request) {
		log := getCommitLog(repo._Repo, master.Reference)

		render(w, "log",
			struct {
				Repo    Repo
				Commits []Commit
			}{
				Repo:    repo,
				Commits: log,
			})
	})

	r.HandleFunc("/commit/{hash}/", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		oid, _ := git.NewOid(vars["hash"])

		c, err := repo._Repo.LookupCommit(oid)
		if err != nil {
			log.Fatal("unable to find commit ", oid, ":", err)
		}
		defer c.Free()

		diff := getCommitDiff(repo._Repo, c)

		render(w, "diff", struct {
			Repo Repo
			Diff Diff
		}{
			repo,
			diff,
		})
	})

	r.HandleFunc("/blob/{filepath:.*}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		s, err := readBlob(repo._Repo, firstCommit, vars["filepath"])
		if err != nil {
			s = err.Error()
		}

		render(w, "file", struct {
			Repo Repo
			Text string
		}{
			repo,
			s,
		})
	})

	r.HandleFunc("/tree/{filepath:.*}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		render.renderFileTree(w, repo, firstCommit, vars["filepath"])
	})

	http.Handle("/", r)
	http.ListenAndServe(":8000", nil)
}
