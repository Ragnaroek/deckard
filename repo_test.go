package deckard

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDiffRepo(t *testing.T) {
	tc := []struct {
		desc         string
		diffStr      string
		expectedDiff *Diff
	}{
		{
			desc:         "empty diff",
			diffStr:      "",
			expectedDiff: &Diff{},
		},
		{
			desc:    "small diff",
			diffStr: "14      0       repo.go\n1       16      slat.go",
			expectedDiff: &Diff{
				[]NumStat{{14, 0, "repo.go"}, {1, 16, "slat.go"}},
			},
		},
		{
			desc:    "diff with extra newline at the end",
			diffStr: "14      0       repo.go\n1       16      slat.go\n",
			expectedDiff: &Diff{
				[]NumStat{{14, 0, "repo.go"}, {1, 16, "slat.go"}},
			},
		},
		{
			desc:    "move commit",
			diffStr: "0      0       services/{foo => echo}/Makefile\n1       16      slat.go",
			expectedDiff: &Diff{
				[]NumStat{{0, 0, "services/{foo => echo}/Makefile"}, {1, 16, "slat.go"}},
			},
		},
	}
	for _, c := range tc {
		t.Run(c.desc, func(t *testing.T) {
			diff, err := parseNumStat(c.diffStr)
			if err != nil {
				t.Errorf("unexpected error: %#v", err)
			}
			if diff := cmp.Diff(diff, c.expectedDiff); diff != "" {
				t.Errorf("unexpected diff: %s", diff)
			}
		})
	}
}
