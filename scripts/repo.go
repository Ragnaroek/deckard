package main

import (
	"fmt"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func main() {
	repo, err := git.PlainOpen("/Users/mb/pprojects/deckard")
	if err != nil {
		panic(err)
	}

	commits, err := repo.Log(&git.LogOptions{All: true})
	if err != nil {
		panic(err)
	}

	err = commits.ForEach(func(commit *object.Commit) error {
		parent, err := commit.Parent(0)
		if err != nil {
			return err
		}
		patch, err := commit.Patch(parent)
		if err != nil {
			return err
		}

		files := patch.FilePatches()
		for _, file := range files {
			from, to := file.Files()
			if from != nil {
				fmt.Printf("from = %#v, ", from.Path())
			} else {
				fmt.Printf("from = nil, ")
			}
			if to != nil {
				fmt.Printf("to = %#v\n", to.Path())
			} else {
				fmt.Printf("to = nil\n")
			}
		}

		fmt.Println()
		return nil
	})
	if err != nil {
		panic(err)
	}
}
