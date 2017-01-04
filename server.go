package main

import "net/http"
import "html/template"
import "fmt"
import "io/ioutil"
import "log"
import "strings"
import "bytes"
import "github.com/libgit2/git2go"
import "github.com/dustin/go-humanize"
import "github.com/gorilla/mux"

type RepoMeta struct {
	Name        string
	Url         string
	Description string
}

func renderPage(repo *git.Repository, w http.ResponseWriter, t *template.Template, content template.HTML) {
	desc, _ := ioutil.ReadFile("../dirt/.git/description")
	t.ExecuteTemplate(w, "main", struct {
		Meta    RepoMeta
		Content template.HTML
	}{
		RepoMeta{
			Name:        "test",
			Url:         "",
			Description: string(desc),
		},
		content,
	})
}

// TODO: get commit log for an arbitrary branch
func getCommitLog(repo *git.Repository) []*git.Commit {
	r, err := repo.Walk()
	if err != nil {
		log.Print("failed to walk repo: ", err)
	}

	masterObj, err := repo.RevparseSingle("master")
	if err != nil {
		log.Print("failed to get master: ", err)
	}

	r.Push(masterObj.Id())
	r.Sorting(git.SortTime)
	r.SimplifyFirstParent()

	id := &(git.Oid{})

	commits := []*git.Commit{}
	for r.Next(id) == nil {
		c, _ := repo.LookupCommit(id)
		commits = append(commits, c)
	}
	return commits
}

func getCommitPatches(repo *git.Repository, commit *git.Commit) []string {
	// TODO check for multiple parent commits
	p, _ := commit.Parent(0).Tree()
	c, _ := commit.Tree()
	o, _ := git.DefaultDiffOptions()
	//r := bytes.NewBufferString("")
	diff, _ := repo.DiffTreeToTree(p, c, &o)
	defer diff.Free()

	n, _ := diff.NumDeltas()
	var r []string
	for i := 0; i < n; i++ {
		patch, _ := diff.Patch(i)

		s, _ := patch.String()
		r = append(r, s)

		patch.Free()
	}
	return r
}

func readFile(repo *git.Repository, filename string) (string, error) {
	masterObj, err := repo.RevparseSingle("master")
	if err != nil {
		log.Print("failed to get master: ", err)
	}

	c, err := masterObj.AsCommit()
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

func getFileTree(repo *git.Repository) []string {
	masterObj, _ := repo.RevparseSingle("master")
	c, _ := masterObj.AsCommit()
	t, _ := c.Tree()
	var r []string
	for i := uint64(0); i < t.EntryCount(); i++ {
		r = append(r, t.EntryByIndex(i).Name)
	}
	return r
}

func main() {
	repoPath := "../dirt/"
	repo, err := git.OpenRepository(repoPath)
	defer repo.Free()
	if err != nil {
		log.Print("failed to open repo: ", err)
	}

	templates := template.Must(template.ParseGlob("t/*"))

	if err != nil {
		log.Print("couldn't parse templates: ", err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ft := getFileTree(repo)

		html_buffer := bytes.NewBufferString("")
		fmt.Fprintf(html_buffer, "<table>")
		for _, v := range ft {
			fmt.Fprintf(html_buffer, "<tr><td><a href='/blob/%s'>%s</a></tr>", v, v)
		}
		fmt.Fprintf(html_buffer, "</table>")
		renderPage(repo, w, templates, template.HTML(html_buffer.String()))
	})

	r.HandleFunc("/log/", func(w http.ResponseWriter, r *http.Request) {
		// TODO: figure out how to render directly to the response writer rather than returning a string
		log := getCommitLog(repo)
		html_buffer := bytes.NewBufferString("")
		fmt.Fprintf(html_buffer, "<table>")
		for _, v := range log {
			fmt.Fprintf(html_buffer, "<tr><td><a href='/commit/%s/'>%s</a></td><td>%s</td><td>%s</td></tr>", v.Id().String(), v.Summary(), v.Committer().Name, humanize.Time(v.Committer().When))
		}
		fmt.Fprintf(html_buffer, "</table>")
		renderPage(repo, w, templates, template.HTML(html_buffer.String()))
	})

	r.HandleFunc("/commit/{hash}/", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		oid, _ := git.NewOid(vars["hash"])

		c, _ := repo.LookupCommit(oid)
		defer c.Free()

		d := getCommitPatches(repo, c)

		renderPage(repo, w, templates, template.HTML("<h1>Diff</h1><pre>"+strings.Join(d, "\n")+"</pre>"))
	})

	r.HandleFunc("/blob/{filename}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		s, err := readFile(repo, vars["filename"])
		if err != nil {
			s = err.Error()
		}

		renderPage(repo, w, templates, template.HTML("<h1>"+vars["filename"]+"</h1><pre>"+s+"</pre>"))
	})

	http.Handle("/", r)
	http.ListenAndServe(":8000", nil)
}
