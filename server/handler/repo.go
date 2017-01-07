package handler

import (
	"github.com/charles-l/gitamite"

	"github.com/labstack/echo"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
)

func CreateRepo(c echo.Context) error {
	blob, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		return err
	}

	var a gitamite.AuthRequest
	err = json.Unmarshal(blob, &a)
	if err != nil {
		return err
	}

	if len(a.Signature) == 0 || a.Data == nil {
		return fmt.Errorf("Need data and signature")
	}

	if err := a.VerifyRequest(); err != nil {
		return fmt.Errorf("Invalid signature")
	}

	log.Printf("creating new repo: %s", a.Data.(map[string]interface{})["Name"])
	return nil
}
