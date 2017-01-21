package model

import (
	"github.com/libgit2/git2go"
	"path/filepath"
)

type Ref struct {
	*git.Reference
}

func (r Ref) NiceName() string {
	return filepath.Base(r.Name())
}
