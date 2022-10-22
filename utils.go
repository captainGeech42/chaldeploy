package main

import (
	"crypto/sha256"
	"fmt"

	"github.com/captainGeech42/chaldeploy/internal/generic_map"
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

// Cache of hashed values
var hashCache = new(generic_map.MapOf[string, string])

// Compute a non-cryptographic secure hash for a string (uses SHA256)
func HashString(message string) string {
	// check if the hash has already been computed, and return it if it has
	if d, ok := hashCache.Load(message); ok {
		return d
	}

	// hasn't been computed before, hash it
	h := sha256.New()
	h.Write([]byte(message))
	digest := h.Sum(nil)
	d := fmt.Sprintf("%x", digest)[:16]

	// save it in the cache
	hashCache.Store(message, d)

	return d
}
