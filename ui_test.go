package deckard

import (
	"testing"
)

func TestRepoURL(t *testing.T) {
	tc := []struct {
		desc        string
		url         string
		expectedUrl string
	}{
		{
			desc:        "url without username/password",
			url:         "https://github.com/Ragnaroek/iron-wolf",
			expectedUrl: "https://github.com/Ragnaroek/iron-wolf",
		},
		{
			desc:        "url with username/password",
			url:         "https://user:secret@dev.azure.com/my/repo",
			expectedUrl: "https://dev.azure.com/my/repo",
		},
		{
			desc:        "Azure Git url",
			url:         "git@ssh.dev.azure.com:v3/org/project/repo",
			expectedUrl: "https://dev.azure.com/org/project/_git/repo",
		},
	}

	for _, c := range tc {
		t.Run(c.desc, func(t *testing.T) {
			result, err := sanitizeRepoURL(c.url)
			if err != nil {
				t.Errorf("expected no error, but got %#v", err)
			}
			if result != c.expectedUrl {
				t.Errorf("expected %s, got %s", c.expectedUrl, result)
			}
		})
	}
}
