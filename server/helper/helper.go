package helper

// dunno if this is the best place for this functionality, but i don't have
// anywhere better to put it atm.

import (
	"github.com/charles-l/gitamite"
	"github.com/charles-l/gitamite/server/context"

	"github.com/labstack/echo"
	"github.com/libgit2/git2go"

	"fmt"
	"path"
)

func defaultCommit(r *gitamite.Repo, ref *gitamite.Ref) (*gitamite.Commit, error) {
	commitObj, err := ref.Peel(git.ObjectCommit)
	if err != nil {
		return nil, err
	}

	gcommit, err := commitObj.AsCommit()
	if err != nil {
		return nil, err
	}

	return &gitamite.Commit{gcommit}, nil
}

func Repo(c echo.Context) (*gitamite.Repo, error) {
	repo := c.(*server.Context).Repos[c.Param("repo")]
	if repo == nil {
		return nil, fmt.Errorf("no such repo")
	}
	return repo, nil
}

func Ref(c echo.Context) (*gitamite.Ref, error) {
	repo, _ := Repo(c)
	refstr := c.Param("ref")
	if refstr == "" {
		refstr = "master"
	}
	ref, err := repo.LookupRef(refstr)
	return &ref, err
}

// sanatizes the path: don't every call Param("*") directly
func PathParam(c echo.Context) string {
	p := path.Clean(c.Param("*"))
	if p == "" || p == "." {
		p = "/"
	}
	return p
}

func Commit(c echo.Context) (*gitamite.Commit, error) {
	repo, _ := Repo(c)
	var commit *gitamite.Commit
	commitstr := c.Param("commit")
	if commitstr == "" {
		ref, err := Ref(c)
		if err != nil {
			return nil, err
		}
		commit, _ = defaultCommit(repo, ref)
	} else {
		var err error
		commit, err = repo.LookupCommit(commitstr)
		if err != nil {
			return nil, err
		}
	}
	return commit, nil
}
