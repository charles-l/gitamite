package gitamite

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/libgit2/git2go"

	"github.com/boltdb/bolt"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"

	"crypto/md5"
	"encoding/hex"
	"html"
	"html/template"

	// syntax highlighting with pygments
	"github.com/charles-l/pygments"
)

var db *bolt.DB

func InitDB() *bolt.DB {
	var err error
	db, err = bolt.Open("/tmp/gitamite.db", 0666, nil)
	if err != nil {
		log.Fatal(err)
	}
	db.Update(func(tx *bolt.Tx) error {
		tx.CreateBucketIfNotExists([]byte("blobCache"))
		return nil
	})
	return db
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

type DiffHunk struct {
	OldPath string
	NewPath string
	Lines   []git.DiffLine
	*git.DiffHunk
}

// TODO: make it generate a quick patch that could be curled
func (h *DiffHunk) AsPatch() *Blob {
	var data [][]byte
	for _, l := range h.Lines {
		data = append(data, []byte(l.Content))
	}
	return &Blob{h.NewPath, "", data}
}

type Diff struct {
	CommitA *Commit
	CommitB *Commit
	Stats   string
	Hunks   []*DiffHunk
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
	diff, _ := repo.DiffTreeToTree(treeB, treeA, &o)

	// TODO: use a struct
	stats, _ := diff.Stats()
	statsStr, _ := stats.String(git.DiffStatsFull, 80)

	r := Diff{
		commitA,
		commitB,
		statsStr,
		nil,
		diff,
	}

	numDiffs := 0
	numAdded := 0
	numDeleted := 0

	var hunks []*DiffHunk
	diff.ForEach(func(file git.DiffDelta, progress float64) (git.DiffForEachHunkCallback, error) {
		numDiffs++

		switch file.Status {
		case git.DeltaAdded:
			numAdded++
		case git.DeltaDeleted:
			numDeleted++
		}

		var hunk DiffHunk

		hunk.OldPath = file.OldFile.Path
		hunk.NewPath = file.NewFile.Path

		hunks = append(hunks, &hunk)
		return func(ghunk git.DiffHunk) (git.DiffForEachLineCallback, error) {
			hunk.DiffHunk = &ghunk
			return func(line git.DiffLine) error {
				hunk.Lines = append(hunk.Lines, line)
				return nil
			}, nil
		}, nil
	}, git.DiffDetailLines)

	r.Hunks = hunks
	return r
}

type Commit struct {
	User *User
	*git.Commit
}

func MakeCommit(g *git.Commit) *Commit {
	return &Commit{GetUserFromEmail(g.Committer().Email),
		g}
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

	return MakeCommit(c), nil
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
		c := MakeCommit(g)
		commits = append(commits, *c)
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
	p, err := GetConfigValue("pubkeyring_path")
	if err != nil {
		return nil
	}
	keys, _ := ReadKeyringFile(p)

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

type Blob struct {
	Path string
	Type string
	Data [][]byte
}

func (b *Blob) ByteArray() []byte {
	return bytes.Join(b.Data, []byte(""))
}

type Blame struct {
	Users []*User
	*Blob
}

func (repo *Repo) ReadBlob(commit *Commit, filepath string) (*Blob, error) {
	t, err := commit.Tree()
	if err != nil {
		log.Print("invalid tree: ", err)
	}

	te, _ := t.EntryByPath(filepath)
	if te == nil {
		return nil, fmt.Errorf("no such file/blob/tree entry %s", filepath)
	}

	f, err := repo.Lookup(te.Id)
	if err != nil {
		return nil, err
	}

	b, err := f.AsBlob()
	if err != nil {
		return nil, err
	}

	ext := path.Ext(filepath)
	if ext != "" {
		ext = ext[1:]
	}
	return &Blob{filepath, ext, bytes.SplitAfter(b.Contents(), []byte("\n"))}, nil
}

// TODO: cache this
func (repo *Repo) ReadBlobBlame(commit *Commit, filepath string) (*Blame, error) {
	o, _ := git.DefaultBlameOptions()
	o.NewestCommit = commit.Id()
	blame, err := repo.BlameFile(filepath, &o)
	if err != nil {
		return nil, err
	}
	blob, err := repo.ReadBlob(commit, filepath)
	b := Blame{[]*User{}, blob}

	// TODO: handle Windows line endings
	for i, l := range b.Data {
		hunk, err := blame.HunkByLine(i)
		if err != nil {
			// TODO: FIXME: a quick 'n' dirty hack
			continue
		}
		b.Data = append(b.Data, l)
		b.Users = append(b.Users, GetUserFromEmail(hunk.FinalSignature.Email))
	}
	return &b, nil
}

// TODO: possibly do this for known blobs in a separate thread when staring the server?
func HighlightedBlobHTML(b *Blob) template.HTML {
	m := md5.New()
	m.Write(b.ByteArray())
	k := []byte("blob:" + hex.EncodeToString(m.Sum(nil)))

	var htmlBlob []byte

	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("blobCache"))
		if e := b.Get(k); e != nil {
			htmlBlob = e
		}
		return nil
	})

	if htmlBlob != nil {
		return template.HTML(string(htmlBlob))
	}

	h, err := pygments.Highlight(b.ByteArray(), b.Type, "html", "utf-8")
	if err != nil {
		h = "<pre>" + html.EscapeString(string(b.ByteArray())) + "</pre>"
	}
	r := template.HTML(h)

	// theoretically this is safe in a goroutine
	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("blobCache"))
		b.Put(k, []byte(h))
		return nil
	})

	return r
}
