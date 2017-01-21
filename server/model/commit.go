package model

import (
	"github.com/libgit2/git2go"
	"time"
)

type Commit struct {
	User *User
	*git.Commit
}

func MakeCommit(g *git.Commit) *Commit {
	return &Commit{UserFromEmail(g.Committer().Email), g}
}

func (c Commit) Hash() string {
	return c.Id().String()
}

func (c Commit) Date() time.Time {
	return c.Author().When
}
