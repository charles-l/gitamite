package context

import (
	"github.com/charles-l/gitamite/server/model"
	"github.com/labstack/echo"
)

type Context struct {
	echo.Context
	Repos map[string]*model.Repo
}
