package model

import (
	"github.com/libgit2/git2go"
	"log"

	"path/filepath"
)

type TreeEntry struct {
	DirPath string
	*git.TreeEntry
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
