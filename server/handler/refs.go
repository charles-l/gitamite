package handler

import (
	"github.com/charles-l/gitamite"
	"github.com/charles-l/gitamite/server/helper"

	"github.com/labstack/echo"
	"github.com/libgit2/git2go"

	"net/http"
)

func Refs(c echo.Context) error {
	repo, _ := helper.Repo(c)

	iter, _ := repo.NewBranchIterator(git.BranchLocal)

	var refs []gitamite.Ref
	iter.ForEach(func(b *git.Branch, t git.BranchType) error {
		refs = append(refs, gitamite.Ref{b.Reference})
		return nil
	})

	c.Render(http.StatusOK, "refs", struct {
		Repo *gitamite.Repo
		Refs []gitamite.Ref
	}{
		repo,
		refs,
	})
	return nil
}
