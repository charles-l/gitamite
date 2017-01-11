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
	"net/http"
	"os"
	"path"
)

// TODO: Move to library
func readAuthJSONRequest(c echo.Context) (*gitamite.AuthRequest, error) {
	blob, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		return nil, err
	}

	var a gitamite.AuthRequest
	err = json.Unmarshal(blob, &a)
	if err != nil {
		return nil, err
	}

	if len(a.Signature) == 0 || a.Data == nil {
		return nil, fmt.Errorf("Need data and signature")
	}

	if err := a.VerifyRequest(); err != nil {
		return nil, fmt.Errorf("Invalid signature")
	}
	return &a, nil
}

func exists(filepath string) bool {
	if stat, err := os.Stat(filepath); err == nil && stat.IsDir() {
		return true
	}
	return false
}

func DeleteRepo(c echo.Context) error {
	a, err := readAuthJSONRequest(c)
	if err != nil {
		return err
	}

	name, ok := a.Data.(map[string]interface{})["Name"].(string)
	name = path.Clean(name) // sanatize

	if !ok {
		return fmt.Errorf("need a valid repo name")
	}

	p, err := gitamite.GetConfigValue("repo_dir")
	if err != nil {
		return err
	}

	repoPath := path.Join(p, name)
	if path.Dir(repoPath) == "/" || repoPath == "/" {
		log.Fatal("cowardly bailing out 'cause I don't want to accidentally delete something important: " + repoPath)
	}
	if !exists(repoPath) {
		return fmt.Errorf("repo doesn't exist")
	}

	log.Printf("deleting repo %s", repoPath)
	os.RemoveAll(repoPath)
	delete(c.(*server.Context).Repos, name)
	return nil
}

func CreateRepo(c echo.Context) error {
	a, err := readAuthJSONRequest(c)
	if err != nil {
		return err
	}

	name, ok := a.Data.(map[string]interface{})["Name"].(string)
	name = path.Clean(name) // sanatize

	if !ok {
		return fmt.Errorf("need a valid repo name")
	}

	p, err := gitamite.GetConfigValue("repo_dir")
	if err != nil {
		return err
	}

	newRepoPath := path.Join(p, name)
	if exists(newRepoPath) {
		return fmt.Errorf("repo already exists")
	}

	log.Printf("creating new repo: %s", newRepoPath)

	repo, err := git.InitRepository(newRepoPath, true)
	if err != nil {
		return err
	}
	c.(*server.Context).Repos[name] = &gitamite.Repo{name, newRepoPath, "", repo}
	return nil
}

func Repos(c echo.Context) error {
	repos := c.(*server.Context).Repos

	vals := make([]*gitamite.Repo, 0, len(repos))
	for _, v := range repos {
		vals = append(vals, v)
	}

	c.Render(http.StatusOK, "repos", struct {
		Repo  *gitamite.Repo
		Repos []*gitamite.Repo
	}{
		nil,
		vals,
	})
	return nil
}
