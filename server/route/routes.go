package route

import (
	"github.com/charles-l/gitamite/server/handler"
	"github.com/charles-l/gitamite/server/model"
	"github.com/labstack/echo"

	"path"
)

func Setup(e *echo.Echo) {
	e.Static("/a", "pub")

	e.GET("/", handler.Repos)

	e.GET("/repo/:repo", handler.FileTree)
	e.GET("/repo/:repo/refs", handler.Refs)

	e.GET("/repo/:repo/commits", handler.FullCommits)
	e.GET("/repo/:repo/:ref/commits", handler.Commits)

	e.GET("/repo/:repo/blob/*", handler.File)
	e.GET("/repo/:repo/blame/*", handler.Blame)

	//TODO: add blame version of this
	e.GET("/repo/:repo/commit/:commit/blob/*", handler.File)

	e.GET("/repo/:repo/tree/*", handler.FileTree)
	e.GET("/repo/:repo/commit/:commit/tree/*", handler.FileTree)

	e.GET("/repo/:repo/commit/:oidA", handler.Diff)

	e.POST("/repo", handler.CreateRepo)
	e.DELETE("/repo", handler.DeleteRepo)

	e.GET("/user/:email", handler.User)
}

func RepoPath(r *model.Repo) string {
	return path.Join("/", "repo", r.Name)
}

// TODO: clean this up and make CommitPath do the logic to
// determine if commit is part of url
func CommitPath(r *model.Repo, c *model.Commit) string {
	return path.Join(RepoPath(r), "commit", c.Hash())
}

func CommitsPath(r *model.Repo, e *model.Ref) string {
	if e == nil {
		return path.Join(RepoPath(r), "commits")
	} else {
		return path.Join(RepoPath(r), e.Name(), "commits")
	}
}

func BlobPath(r *model.Repo, c *model.Commit, b *model.Blob) string {
	if c == nil {
		return path.Join(RepoPath(r), "blob", b.Path)
	} else {
		return path.Join(CommitPath(r, c), "blob", b.Path)
	}
}

func BlamePath(r *model.Repo, b *model.Blob) string {
	return path.Join(RepoPath(r), "blame", b.Path)
}

func UserPath(u *model.User) string {
	return path.Join("/", "user", u.Email)
}

//TODO:
//func TreePath(r *model.Repo, c *model.Commit, )
