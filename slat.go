package deckard

import (
	"strings"
)

// slatScore calculates a slat (_s_hould-_l_ook-_a_t-i_t_) score between 0 and 100.0.
// 100.0 you definitely need to look into it, 0.0 means there was nothing harmful detected in the
// commit.
func slatScore(diff *Diff) (int, error) {

	for _, stat := range diff.Stats {
		if touchesGoModules(stat.File) {
			return 100.0, nil
		}
	}

	return 0.0, nil
}

func touchesGoModules(file string) bool {
	return strings.Contains(file, "go.mod") || strings.Contains(file, "go.sum")
}
