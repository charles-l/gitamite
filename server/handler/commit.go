package handler

import (
	"net/http"

	"github.com/charles-l/gitamite"
	"github.com/charles-l/gitamite/server/context"
	"github.com/labstack/echo"
)

func parseRef(r *gitamite.Repo, refstr string) (gitamite.Ref, error) {
	if refstr == "" {
		refstr = "master"
	}
	ref, err := r.LookupRef(refstr)
	return ref, err
}

func CommitsHandler(c echo.Context) error {
	ref, err := parseRef(&c.(*server.Context).Repo, c.Param("ref"))
	if err != nil {
		return err
	}

	log := gitamite.GetCommitLog(&c.(*server.Context).Repo, ref)

	c.Render(http.StatusOK, "log",
		struct {
			Repo    *gitamite.Repo
			Commits []gitamite.Commit
		}{
			&c.(*server.Context).Repo,
			log,
		})
	return nil
}
