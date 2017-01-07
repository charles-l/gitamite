package server

import (
	"github.com/charles-l/gitamite"
	"github.com/labstack/echo"
)

type Context struct {
	echo.Context
	Repo gitamite.Repo
}
