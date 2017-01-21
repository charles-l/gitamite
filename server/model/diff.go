package model

import (
	"github.com/libgit2/git2go"
)

type DiffHunk struct {
	OldPath string
	NewPath string
	Lines   []git.DiffLine
	*git.DiffHunk
}

// TODO: make it generate a quick patch that could be curled
func (h *DiffHunk) AsPatch() *Blob {
	var data [][]byte
	for _, l := range h.Lines {
		data = append(data, []byte(l.Content))
	}
	return &Blob{h.NewPath, "", data}
}

type Diff struct {
	CommitA *Commit
	CommitB *Commit
	Stats   string
	Hunks   []*DiffHunk
	*git.Diff
}

func GetDiff(repo *Repo, commitA *Commit, commitB *Commit) Diff {
	treeA, _ := commitA.Tree()
	var treeB *git.Tree
	if commitB == nil {
		treeB = nil
	} else {
		treeB, _ = commitB.Tree()
	}
	o, _ := git.DefaultDiffOptions()
	diff, _ := repo.DiffTreeToTree(treeB, treeA, &o)

	// TODO: use a struct
	stats, _ := diff.Stats()
	statsStr, _ := stats.String(git.DiffStatsFull, 80)

	r := Diff{
		commitA,
		commitB,
		statsStr,
		nil,
		diff,
	}

	numDiffs := 0
	numAdded := 0
	numDeleted := 0

	var hunks []*DiffHunk
	diff.ForEach(func(file git.DiffDelta, progress float64) (git.DiffForEachHunkCallback, error) {
		numDiffs++

		switch file.Status {
		case git.DeltaAdded:
			numAdded++
		case git.DeltaDeleted:
			numDeleted++
		}

		var hunk DiffHunk

		hunk.OldPath = file.OldFile.Path
		hunk.NewPath = file.NewFile.Path

		hunks = append(hunks, &hunk)
		return func(ghunk git.DiffHunk) (git.DiffForEachLineCallback, error) {
			hunk.DiffHunk = &ghunk
			return func(line git.DiffLine) error {
				hunk.Lines = append(hunk.Lines, line)
				return nil
			}, nil
		}, nil
	}, git.DiffDetailLines)

	r.Hunks = hunks
	return r
}
