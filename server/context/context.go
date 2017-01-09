package server

import (
	"github.com/charles-l/gitamite"
	"github.com/labstack/echo"
)

type Context struct {
	echo.Context
	Repos map[string]*gitamite.Repo
}
