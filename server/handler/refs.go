package handler

import (
	"github.com/charles-l/gitamite/server/helper"
	"github.com/charles-l/gitamite/server/model"

	"github.com/labstack/echo"

	"net/http"
)

func Refs(c echo.Context) error {
	repo, err := helper.RepoParam(c)
	if err != nil {
		return err
	}

	refs := repo.Refs()

	c.Render(http.StatusOK, "refs", struct {
		Repo *model.Repo
		Refs []*model.Ref
	}{
		repo,
		refs,
	})
	return nil
}
