package gitamite

import (
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"time"

	"github.com/libgit2/git2go"
)

type TreeEntry struct {
	DirPath string
	*git.TreeEntry
}

type Ref struct {
	*git.Reference
}

func (r Ref) NiceName() string {
	return filepath.Base(r.Name())
}

type Diff struct {
	CommitA *Commit
	CommitB *Commit
	Stats   string
	Patches []string
	*git.Diff
}

type Commit struct {
	*git.Commit
}

type Repo struct {
	Name        string
	Filepath    string
	Description string
	*git.Repository
}

func LoadRepository(name string, path string) *Repo {
	repo, err := git.OpenRepository(path)
	if err != nil {
		log.Fatal("failed to open repo ", path, ":", err)
	}
	desc, err := ioutil.ReadFile(path + "/.git/description")
	if err != nil {
		log.Print("failed to get repo description ", path, ":", err)
		desc = []byte("")
	}
	return &Repo{
		name,
		path,
		string(desc),
		repo,
	}
}

func (r Repo) LookupRef(ref string) (Ref, error) {
	master, err := r.LookupBranch(ref, git.BranchAll)
	if err != nil {
		return Ref{}, fmt.Errorf("failed to fetch ref: " + err.Error())
	}
	return Ref{master.Reference}, nil
}

func (r Repo) DefaultRef() (Ref, error) {
	return r.LookupRef("master")
}

func (r Repo) DefaultCommit() (Commit, error) {
	master, err := r.LookupRef("master")
	if err != nil {
		return Commit{}, err
	}

	commitObj, err := master.Peel(git.ObjectCommit)
	if err != nil {
		return Commit{}, err
	}

	gcommit, err := commitObj.AsCommit()
	if err != nil {
		return Commit{}, err
	}

	return Commit{gcommit}, nil
}

func (r Repo) LookupCommit(hash string) (Commit, error) {
	oid, err := git.NewOid(hash)
	if err != nil {
		return Commit{}, err
	}

	c, err := r.Repository.LookupCommit(oid)
	if err != nil {
		return Commit{}, err
	}

	return Commit{c}, nil
}

func GetCommitLog(repo *Repo, ref Ref) []Commit {
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
		g, _ := repo.Repository.LookupCommit(id)
		c := Commit{g}
		commits = append(commits, c)
	}
	return commits
}

func (c Commit) Hash() string {
	return c.Id().String()
}

func (c Commit) Date() time.Time {
	return c.Author().When
}

func GetDiff(repo *Repo, commitA *Commit, commitB *Commit) Diff {
	treeA, _ := commitA.Tree()
	var treeB *git.Tree
	if commitB == nil {
		treeB = nil
	} else {
		treeB, _ = commitB.Tree()
	}
	o, _ := git.DefaultDiffOptions()
	diff, _ := repo.DiffTreeToTree(treeA, treeB, &o)

	stats, _ := diff.Stats()
	statsStr, _ := stats.String(git.DiffStatsFull, 80)

	r := Diff{
		commitA,
		commitB,
		statsStr,
		[]string{},
		diff,
	}
	n, _ := diff.NumDeltas()
	for i := 0; i < n; i++ {
		patch, _ := diff.Patch(i)

		s, _ := patch.String()
		r.Patches = append(r.Patches, s)
	}
	return r
}

func (repo *Repo) ReadBlob(commit *Commit, filepath string) ([]byte, error) {
	t, err := commit.Tree()
	if err != nil {
		log.Print("invalid tree: ", err)
	}

	te, _ := t.EntryByPath(filepath)
	if te == nil {
		return nil, fmt.Errorf("no such file/blob/tree entry %g", filepath)
	}

	f, err := repo.Lookup(te.Id)
	if err != nil {
		return nil, err
	}

	b, err := f.AsBlob()
	if err != nil {
		return nil, err
	}

	return b.Contents(), nil
}

func GetTreeEntries(t *git.Tree, treePath string) []TreeEntry {
	var r []TreeEntry
	for i := uint64(0); i < t.EntryCount(); i++ {
		r = append(r, TreeEntry{treePath, t.EntryByIndex(i)})
	}
	return r
}

// TODO: combine getTreeEntry and getSubTree into one function
func GetSubTree(t *git.Tree, treePath string) []TreeEntry {
	subentry, _ := t.EntryByPath(treePath)
	if subentry.Type != git.ObjectTree {
		log.Fatal("path is not a subtree ", treePath, " - is ", subentry.Type)
	}
	subtree, _ := t.Object.Owner().LookupTree(subentry.Id)
	return append([]TreeEntry{getParentDir(treePath)}, GetTreeEntries(subtree, treePath)...)
}

func GetTreeEntry(t *git.Tree, treePath string) TreeEntry {
	e, _ := t.EntryByPath(treePath)
	return TreeEntry{treePath, e}
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
