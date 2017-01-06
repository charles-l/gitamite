package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/gorilla/mux"
	"github.com/libgit2/git2go"
	"golang.org/x/crypto/openpgp"
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
	_CommitA *git.Commit
	_CommitB *git.Commit
	_Diff    *git.Diff
	Stats    string
	Patches  []string
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

func getDiff(repo *git.Repository, commitA *git.Commit, commitB *git.Commit) Diff {
	// TODO check for multiple parent commits
	treeA, _ := commitA.Tree()
	treeB, _ := commitB.Tree()
	o, _ := git.DefaultDiffOptions()
	diff, _ := repo.DiffTreeToTree(treeA, treeB, &o)
	defer diff.Free()

	stats, _ := diff.Stats()
	statsStr, _ := stats.String(git.DiffStatsFull, 80)

	r := Diff{
		_CommitA: commitA,
		_CommitB: commitB,
		_Diff:    diff,
		Stats:    statsStr,
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

const pubRing = "/home/nc/.gnupg/pubring.gpg"

type AuthRequest struct {
	Signature, Data []byte
}

func (r AuthRequest) verifyRequest() error {
	f, err := ioutil.ReadFile(pubRing)
	if err != nil {
		return err
	}
	keyring, err := openpgp.ReadKeyRing(bytes.NewReader(f))
	if err != nil {
		return err
	}

	if _, err = openpgp.CheckArmoredDetachedSignature(keyring,
		bytes.NewReader(r.Data),
		bytes.NewReader(r.Signature)); err != nil {
		return err
	}
	return nil
}

type CreateHandler struct {
	Render Renderer
}

func (h CreateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	blob, err := ioutil.ReadAll(r.Body)

	var a AuthRequest
	json.Unmarshal(blob, &a)

	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if len(a.Signature) == 0 || len(a.Data) == 0 {
		http.Error(w, "Need data and signature", 400)
		return
	}

	if err := a.verifyRequest(); err == nil {
		fmt.Fprintf(w, "YUUSS!")
	} else {
		http.Error(w, "Not the password", 401)
	}
}

type CommitLogHandler struct {
	Render Renderer
	Repo   Repo
	Ref    *git.Reference
}

func (h CommitLogHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := getCommitLog(h.Repo._Repo, h.Ref)

	h.Render(w, "log",
		struct {
			Repo    Repo
			Commits []Commit
		}{
			h.Repo,
			log,
		})
}

type FileTreeHandler struct {
	Render Renderer
	Repo   Repo
	Commit *git.Commit
}

func (h FileTreeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := "/"
	if vars := mux.Vars(r); vars["path"] != "" {
		path = vars["path"]
	}
	h.Render.renderFileTree(w, h.Repo, h.Commit, path)
}

type DiffHandler struct {
	Render Renderer
	Repo   Repo
}

func (h DiffHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	oidA, _ := git.NewOid(vars["oidA"])

	commitA, err := h.Repo._Repo.LookupCommit(oidA)
	if err != nil {
		http.Error(w, "unable to find commit", 404)
	}
	defer commitA.Free()

	var commitB *git.Commit
	if vars["oidB"] == "" {
		commitB = commitA.Parent(0)
	} else {
		oidB, _ := git.NewOid(vars["oidB"])
		commitB, err = h.Repo._Repo.LookupCommit(oidB)
		if err != nil {
			http.Error(w, "unable to find commit", 404)
		}
	}

	diff := getDiff(h.Repo._Repo, commitA, commitB)

	h.Render(w, "diff", struct {
		Repo Repo
		Diff Diff
	}{
		h.Repo,
		diff,
	})
}

type FileHandler struct {
	Render Renderer
	Repo   Repo
	Commit *git.Commit
}

func (h FileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	s, err := readBlob(h.Repo._Repo, h.Commit, vars["path"])
	if err != nil {
		http.Error(w, "file not found", 404)
	}

	h.Render(w, "file", struct {
		Repo Repo
		Text string
	}{
		h.Repo,
		s,
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
	r.Handle("/", FileTreeHandler{render, repo, firstCommit})
	r.Handle("/create", CreateHandler{render})
	r.Handle("/log/", CommitLogHandler{render, repo, master.Reference})
	r.Handle("/commit/{oidA}/", DiffHandler{render, repo})
	r.Handle("/blob/{path:.*}", FileHandler{render, repo, firstCommit})
	r.Handle("/tree/{path:.*}", FileTreeHandler{render, repo, firstCommit})

	http.Handle("/", r)
	http.ListenAndServe(":8000", nil)
}
