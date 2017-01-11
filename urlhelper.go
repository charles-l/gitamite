package gitamite

import (
	"github.com/libgit2/git2go"
	"path"
	"path/filepath"
)

type URLable interface {
	URL() string
}

func (r Repo) URL() string {
	return path.Join("/", "repo", r.Name)
}

func (r Commit) URL() string {
	return r.Object.Id().String()
}

func (r Ref) URL() string {
	return r.Name()
}

func (t TreeEntry) URL() string {
	if t.Type == git.ObjectBlob {
		return filepath.Join("blob", t.DirPath, t.Name)
	} else if t.Type == git.ObjectTree {
		if t.DirPath == "" {
			return "/"
		} else {
			if t.Name == ".." { // TODO: simplify
				return filepath.Join("tree", t.DirPath)
			} else {
				return filepath.Join("tree", t.DirPath, t.Name)
			}
		}
	}
	return ""
}

func (u User) URL() string {
	return path.Join("/", "user", u.Email)
}
