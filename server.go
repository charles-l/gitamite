package main

import "net/http"
import "io"
import "html/template"
import "fmt"
import "io/ioutil"
import "path/filepath"
import "log"
import "strings"
import "time"
import "github.com/libgit2/git2go"
import "github.com/dustin/go-humanize"
import "github.com/gorilla/mux"

type RepoMeta struct {
	Name        string
	URL         string
	Description string
}

type TrackedFile struct {
	Filename   string
	Object     *git.Object
	LastCommit *git.Commit
}

type Commit struct {
	GCommit *git.Commit
	Message string
	Author  string
	Hash    string
	Date    time.Time
}

type Diff struct {
	GCommitA    *git.Commit
	GCommitB    *git.Commit
	CommitAHash string
	CommitBHash string
	Stats       string
	Patches     []string
}

// TODO: get commit log for an arbitrary branch
func getCommitLog(repo *git.Repository, obj *git.Object) []Commit {
	r, err := repo.Walk()
	if err != nil {
		log.Print("failed to walk repo: ", err)
	}

	r.Push(obj.Id())
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
		GCommitA:    commit.Parent(0),
		GCommitB:    commit,
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

func readFile(repo *git.Repository, commitObj *git.Object, filename string) (string, error) {
	c, err := commitObj.AsCommit()
	if err != nil {
		log.Print("invalid commit: ", err)
	}

	t, err := c.Tree()
	if err != nil {
		log.Print("invalid tree: ", err)
	}

	te := t.EntryByName(filename)
	if te == nil {
		log.Print("no file: ", filename)
		return "", fmt.Errorf("no such file/blob/tree entry %g", filename)
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

func getFileTreeForCommit(commitObj *git.Object) []TrackedFile {
	commit, _ := commitObj.AsCommit()
	t, _ := commit.Tree()
	var r []TrackedFile
	for i := uint64(0); i < t.EntryCount(); i++ {
		e := t.EntryByIndex(i)
		r = append(r, TrackedFile{
			Filename:   e.Name,
			Object:     commitObj,
			LastCommit: commit, // FIXME: this is wrong
		})
	}
	return r
}

func createPageRenderer() func(w io.Writer, name string, i interface{}) {
	templateFuncs := template.FuncMap{
		"humanizeTime": func(t time.Time) string {
			return humanize.Time(t)
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

func main() {
	renderPage := createPageRenderer()
	desc, _ := ioutil.ReadFile("../dirt/.git/description")
	repoMeta := RepoMeta{
		Name:        "test",
		URL:         "",
		Description: string(desc),
	}

	repoPath := "../dirt/"
	repo, err := git.OpenRepository(repoPath)
	defer repo.Free()
	if err != nil {
		log.Print("failed to open repo: ", err)
	}

	masterObj, err := repo.RevparseSingle("master")

	r := mux.NewRouter()
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ft := getFileTreeForCommit(masterObj)

		renderPage(w, "filelist",
			struct {
				Meta  RepoMeta
				Files []TrackedFile
			}{
				repoMeta,
				ft,
			})
	})

	r.HandleFunc("/log/", func(w http.ResponseWriter, r *http.Request) {
		// TODO: figure out how to render directly to the response writer rather than returning a string
		log := getCommitLog(repo, masterObj)

		renderPage(w, "log",
			struct {
				Meta    RepoMeta
				Commits []Commit
			}{
				Meta:    repoMeta,
				Commits: log,
			})
	})

	r.HandleFunc("/commit/{hash}/", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		oid, _ := git.NewOid(vars["hash"])

		c, err := repo.LookupCommit(oid)
		if err != nil {
			log.Fatal("unable to find commit ", oid, ":", err)
		}
		defer c.Free()

		diff := getCommitDiff(repo, c)

		renderPage(w, "diff", struct {
			Meta RepoMeta
			Diff Diff
		}{
			repoMeta,
			diff,
		})
	})

	r.HandleFunc("/blob/{filename}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		s, err := readFile(repo, masterObj, vars["filename"])
		if err != nil {
			s = err.Error()
		}

		renderPage(w, "file", struct {
			Meta RepoMeta
			Text string
		}{
			repoMeta,
			s,
		})
	})

	http.Handle("/", r)
	http.ListenAndServe(":8000", nil)
}
