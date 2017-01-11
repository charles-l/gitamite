package gitamite

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/libgit2/git2go"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
)

func getParentDir(path string) TreeEntry {
	dir, _ := filepath.Split(path)

	g := git.TreeEntry{
		Name: "..",
		Type: git.ObjectTree,
	}

	t := TreeEntry{dir, &g}
	return t
}

type TreeEntry struct {
	DirPath string
	*git.TreeEntry
}

func GetTreeEntries(t *git.Tree, treePath string) []TreeEntry {
	var r []TreeEntry
	for i := uint64(0); i < t.EntryCount(); i++ {
		r = append(r, TreeEntry{treePath, t.EntryByIndex(i)})
	}
	return r
}

// TODO: combine getTreeEntry and getSubTree into one function
func GetSubTree(t *git.Tree, treePath string) ([]TreeEntry, error) {
	subentry, err := t.EntryByPath(treePath)
	if err != nil {
		return nil, err
	}
	if subentry.Type != git.ObjectTree {
		log.Fatal("path is not a subtree ", treePath, " - is ", subentry.Type)
	}
	subtree, _ := t.Object.Owner().LookupTree(subentry.Id)
	return append([]TreeEntry{getParentDir(treePath)}, GetTreeEntries(subtree, treePath)...), nil
}

func GetTreeEntry(t *git.Tree, treePath string) TreeEntry {
	e, _ := t.EntryByPath(treePath)
	return TreeEntry{treePath, e}
}

type Ref struct {
	*git.Reference
}

func (r Ref) NiceName() string {
	return filepath.Base(r.Name())
}

func (r Repo) LookupRef(ref string) (Ref, error) {
	master, err := r.LookupBranch(ref, git.BranchAll)
	if err != nil {
		return Ref{}, fmt.Errorf("failed to fetch ref: " + err.Error())
	}
	return Ref{master.Reference}, nil
}

type Diff struct {
	CommitA *Commit
	CommitB *Commit
	Stats   string
	Patches []string
	*git.Diff
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

type Commit struct {
	*git.Commit
}

func (r Repo) LookupCommit(hash string) (*Commit, error) {
	oid, err := git.NewOid(hash)
	if err != nil {
		return nil, err
	}

	c, err := r.Repository.LookupCommit(oid)
	if err != nil {
		return nil, err
	}

	return &Commit{c}, nil
}

func GetCommitLog(repo *Repo, ref *Ref) []Commit {
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

type Repo struct {
	Name        string
	Filepath    string
	Description string
	*git.Repository
}

func LoadRepository(name string, repoPath string) *Repo {
	repo, err := git.OpenRepository(repoPath)
	if err != nil {
		log.Fatal("failed to open repo ", repoPath, ":", err)
	}
	desc, err := ioutil.ReadFile(path.Join(repoPath, "description"))
	if err != nil {
		log.Print("failed to get repo description ", repoPath, ":", err)
		desc = []byte("")
	}
	return &Repo{
		name,
		repoPath,
		string(desc),
		repo,
	}
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

// TODO: don't return a byte array: return an array of structs
func (repo *Repo) ReadBlobBlame(commit *Commit, filepath string) ([]byte, error) {
	o, _ := git.DefaultBlameOptions()
	o.NewestCommit = commit.Id()
	blame, err := repo.BlameFile(filepath, &o)
	if err != nil {
		return nil, err
	}
	blob, err := repo.ReadBlob(commit, filepath)
	var out [][]byte
	// TODO: handle windows line endings
	for nu, l := range bytes.Split(blob, []byte{'\n'}) {
		hunk, err := blame.HunkByLine(nu + 1)
		if err != nil {
			// TODO: FIXME: a quick 'n' dirty hack
			continue
		}
		out = append(out, bytes.Join([][]byte{
			[]byte(hunk.FinalSignature.Email),
			[]byte(strconv.Itoa(nu + 1)),
			l},
			[]byte("\t")))
	}
	return bytes.Join(out, []byte("\n")), nil
}

type User struct {
	Name   string
	Email  string
	Entity *openpgp.Entity
}

func ArmoredPublicKey(u *User) *bytes.Buffer {
	b := bytes.NewBuffer([]byte{})
	w, err := armor.Encode(b, "PUBLIC KEY BLOCK", map[string]string{})
	if err != nil {
		return nil
	}
	u.Entity.Serialize(w)
	w.Close()
	return b
}

func GetUserFromEmail(email string) *User {
	keys, _ := ReadKeyringFile(GlobalConfig.Auth.PublicKeyring)

	var u User

	for _, e := range keys {
		for k, _ := range e.Identities {
			s := strings.Split(k, "<")
			m := s[1][:len(s[1])-1]
			if m == email {
				u.Name = s[0]
				u.Email = m
				u.Entity = e
			}
		}
	}

	if u == (User{}) {
		return nil
	}
	return &u
}
