package model

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"path"

	"github.com/libgit2/git2go"
)

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

func (r *Repo) LookupRef(ref string) (Ref, error) {
	master, err := r.LookupBranch(ref, git.BranchAll)
	if err != nil {
		return Ref{}, fmt.Errorf("failed to fetch ref: " + err.Error())
	}
	return Ref{master.Reference}, nil
}

func (repo *Repo) Refs() []*Ref {
	iter, _ := repo.NewBranchIterator(git.BranchLocal)

	var refs []*Ref
	iter.ForEach(func(b *git.Branch, t git.BranchType) error {
		refs = append(refs, &Ref{b.Reference})
		return nil
	})
	return refs
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

func (repo *Repo) CommitLog(ref *Ref) []*Commit {
	r, err := repo.Walk()
	if err != nil {
		log.Print("failed to walk repo: ", err)
	}

	if ref == nil {
		r.PushGlob("*")
	} else {
		r.Push(ref.Target())
	}
	r.Sorting(git.SortTime)
	r.SimplifyFirstParent()

	id := &(git.Oid{})

	var commits []*Commit
	for r.Next(id) == nil {
		g, _ := repo.Repository.LookupCommit(id)
		commits = append(commits, MakeCommit(g))
	}
	return commits
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
		b.Users = append(b.Users, UserFromEmail(hunk.FinalSignature.Email))
	}
	return &b, nil
}
