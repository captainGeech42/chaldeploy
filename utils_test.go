package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHash(t *testing.T) {
	assert.Equal(t, "2ba5182aef96aaf7", HashString("hello world what a sweet hash"))
	assert.Equal(t, "e3b0c44298fc1c14", HashString(""))
	assert.Equal(t, "2ba5182aef96aaf7", HashString("hello world what a sweet hash"))
}

func TestContains(t *testing.T) {
	assert.True(t, Contains([]int{1, 2, 3}, 3))
	assert.False(t, Contains([]int{1, 2, 3}, 5))
}
