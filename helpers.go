package gitamite

import (
	"github.com/libgit2/git2go"
	"log"
	"path"
	"path/filepath"
)

type Pathable interface {
	Path() string
}

func (r Repo) Path() string {
	return path.Join("/", "repos", r.Name)
}

func (c Commit) Path() string {
	// use repo path from context
	return path.Join("/repos/gitamite", "commit", c.Hash())
}

func (t TreeEntry) Path() string {
	if t.Type == git.ObjectBlob {
		return filepath.Join("/", "repos", "blah", "blob", t.DirPath, t.Name)
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
