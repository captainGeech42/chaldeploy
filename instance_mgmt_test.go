package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestImageName(t *testing.T) {
	assert.Equal(t, "test-nc", getImageName("captaingeech/test-nc:latest"))
	assert.Equal(t, "ubuntu", getImageName("library.docker.io/_/ubuntu:18.04"))
}
