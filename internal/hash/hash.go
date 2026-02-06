// Package hash provides shared hashing utilities for generating truncated IDs.
package hash

import (
	"crypto/sha256"
	"encoding/hex"
)

// IDLength is the number of hex characters used for truncated hash IDs.
// 16 hex chars = 8 bytes = 64 bits of entropy (sufficient for collision resistance).
const IDLength = 16

// TruncatedSHA256 returns a truncated SHA256 hash of the input string.
// The result is a 16-character hex string.
func TruncatedSHA256(data string) string {
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])[:IDLength]
}

// TruncatedSHA256Bytes returns a truncated SHA256 hash of the input bytes.
// The result is a 16-character hex string.
func TruncatedSHA256Bytes(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])[:IDLength]
}
