package handler

import (
	"github.com/charles-l/gitamite/server/model"
	"github.com/labstack/echo"

	"fmt"
	"net/http"
	"strings"
)

// TODO: memoize this
func User(c echo.Context) error {
	email := c.Param("email")

	u := model.UserFromEmail(email)
	if u == nil {
		return fmt.Errorf("failed to get user: " + email)
	}

	c.String(http.StatusOK, strings.Join([]string{
		u.Name,
		u.Email,
		model.ArmoredPublicKey(u).String(),
	}, "\n"))
	return nil
}
