package handler

import (
	"crypto/rand"
	"fmt"
)

// newUUID generates a random UUID v4 string.
// Uses crypto/rand so no external dependency is required.
func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	// Version 4 (random)
	b[6] = (b[6] & 0x0f) | 0x40
	// Variant 10xx
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf(
		"%08x-%04x-%04x-%04x-%12x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:],
	)
}
