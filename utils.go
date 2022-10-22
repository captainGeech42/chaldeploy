package main

import (
	"crypto/sha256"
	"fmt"
)

// Check if a slice contains a specified element
func Contains[T comparable](haystack []T, needle T) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}

	return false
}

// Compute a hash for a string (uses SHA256)
func HashString(message string) string {
	h := sha256.New()
	h.Write([]byte(message))
	digest := h.Sum(nil)
	return fmt.Sprintf("%x", digest)
}
