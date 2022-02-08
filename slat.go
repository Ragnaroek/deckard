package deckard

import (
	"strings"

	"github.com/go-git/go-git/v5/plumbing/format/diff"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// slatScore calculates a slat (_s_hould-_l_ook-_a_t-i_t_) score between 0 and 100.0.
// 100.0 you definitely need to look into it, 0.0 means there was nothing harmful detected in the
// commit.
func slatScore(c *object.Commit) (int, error) {
	if len(c.ParentHashes) != 1 {
		// always look at merges, or the first commit in the repo
		// TODO how to properly deals with merges?
		return 100.0, nil
	}

	parent, err := c.Parent(0)
	if err != nil {
		return 0, err
	}
	patch, err := c.Patch(parent)
	if err != nil {
		return 0, err
	}

	files := patch.FilePatches()
	for _, file := range files {
		from, to := file.Files()
		if from != nil && touchesGoModules(from) {
			return 100.0, nil
		}
		if to != nil && touchesGoModules(to) {
			return 100.0, nil
		}
	}

	return 0.0, nil
}

func touchesGoModules(diff diff.File) bool {
	return strings.Contains(diff.Path(), "go.mod") || strings.Contains(diff.Path(), "go.sum")
}
