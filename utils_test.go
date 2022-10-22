package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHash(t *testing.T) {
	assert.Equal(t, "2ba5182aef96aaf7f7e71b7ea54ef44f33e047fbd3fe540374f26d7a9ad5c897", HashString("hello world what a sweet hash"))
	assert.Equal(t, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", HashString(""))
}

func TestContains(t *testing.T) {
	assert.True(t, Contains([]int{1, 2, 3}, 3))
	assert.False(t, Contains([]int{1, 2, 3}, 5))
}
