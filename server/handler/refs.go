package handler

import (
	"github.com/charles-l/gitamite"
	"github.com/charles-l/gitamite/server/helper"

	"github.com/labstack/echo"

	"net/http"
)

func Refs(c echo.Context) error {
	repo, err := helper.Repo(c)
	if err != nil {
		return err
	}

	refs := repo.Refs()

	c.Render(http.StatusOK, "refs", struct {
		Repo *gitamite.Repo
		Refs []*gitamite.Ref
	}{
		repo,
		refs,
	})
	return nil
}
