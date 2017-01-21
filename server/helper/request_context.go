package helper

// parses repos, refs, commits, blobs, etc. out of request param

import (
	"github.com/charles-l/gitamite/server/context"
	"github.com/charles-l/gitamite/server/model"

	"github.com/labstack/echo"
	"github.com/libgit2/git2go"

	"fmt"
	"path"
)

func defaultCommit(r *model.Repo, ref *model.Ref) (*model.Commit, error) {
	commitObj, err := ref.Peel(git.ObjectCommit)
	if err != nil {
		return nil, err
	}

	gcommit, err := commitObj.AsCommit()
	if err != nil {
		return nil, err
	}

	return model.MakeCommit(gcommit), nil
}

func RepoParam(c echo.Context) (*model.Repo, error) {
	repo := c.(*context.Context).Repos[c.Param("repo")]
	if repo == nil {
		return nil, fmt.Errorf("no such repo")
	}
	return repo, nil
}

func RefParam(c echo.Context, allowNil bool) (*model.Ref, error) {
	repo, _ := RepoParam(c)
	refstr := c.Param("ref")
	if refstr == "" {
		if allowNil {
			return nil, nil
		} else {
			refstr = "master"
		}
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

func CommitParam(c echo.Context) (*model.Commit, error) {
	repo, _ := RepoParam(c)
	var commit *model.Commit
	commitstr := c.Param("commit")
	if commitstr == "" {
		ref, err := RefParam(c, false)
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
