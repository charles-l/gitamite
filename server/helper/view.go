package helper

import (
	"github.com/charles-l/gitamite"
	"github.com/gosvg/gosvg"
	"github.com/libgit2/git2go"

	"bytes"
)

func RenderLogTree(repo *gitamite.Repo) []byte {
	type commitNode struct {
		x, y   float64
		commit *gitamite.Commit
		branch string
	}

	nodes := make(map[string]*commitNode)

	for i, c := range gitamite.GetCommitLog(repo, nil) {
		y := float64(10 + i*32)
		x := float64(10)
		branchName := ""

		nodes[c.Hash()] = &commitNode{
			x, y, c, branchName,
		}
	}

	var tweakNodes func(phash, hash string)
	tweakNodes = func(chash, hash string) {
		n := nodes[hash]
		if chash != "" {
			if n.branch != "" {
				n.x = nodes[chash].x
			}
			n.branch = nodes[chash].branch
		}
		for i := uint(0); i < n.commit.ParentCount(); i++ {
			tweakNodes(n.commit.Id().String(), n.commit.Parent(i).Id().String())
		}
	}

	refs := repo.Refs()
	for i, r := range refs {
		o, err := r.Branch().Peel(git.ObjectCommit)
		if err != nil {
			continue
		}
		c, _ := o.AsCommit()
		n, _ := r.Branch().Name()

		p := nodes[c.Id().String()]
		p.branch = n
		p.x = float64(10 + i*32)

		tweakNodes("", p.commit.Id().String())
	}

	s := gosvg.NewSVG(float64((len(refs)-1)*32+20), float64((len(nodes)-1)*32+20))
	s.Class = "id=\"commit-graph\""

	for _, n := range nodes {
		s.Circle(n.x, n.y, 4)
		for i := uint(0); i < n.commit.ParentCount(); i++ {
			p := nodes[n.commit.Parent(i).Id().String()]
			l := s.Line(n.x, n.y, p.x, p.y)
			l.Style.Set("stroke-width", "2")
			l.Style.Set("stroke", "black")
		}
	}

	var b bytes.Buffer
	s.Render(&b)

	return b.Bytes()
}
