package deckard

import (
	"math/rand"

	"github.com/go-git/go-git/v5/plumbing/object"
)

func slatScore(c *object.Commit) int {
	// TODO Actually calculate the score
	return int(rand.Float64() * 100.0)
}
