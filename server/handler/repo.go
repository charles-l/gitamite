package handler

import (
	"github.com/charles-l/gitamite"
	"github.com/charles-l/gitamite/server/context"
	"github.com/libgit2/git2go"

	"github.com/labstack/echo"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path"
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

	m := a.Data.(map[string]interface{})
	var name string
	for k, v := range m {
		switch vv := v.(type) {
		case string:
			if k == "Name" {
				name = vv
			}
		}
	}
	if name == "" {
		return fmt.Errorf("need a repository name")
	}

	newRepoPath := path.Join(gitamite.GlobalConfig.RepoDir, name)
	log.Printf("creating new repo: %s", newRepoPath)

	repo, err := git.InitRepository(newRepoPath, true)
	if err != nil {
		return err
	}
	c.(*server.Context).Repos[name] = &gitamite.Repo{name, newRepoPath, "", repo}
	return nil
}
