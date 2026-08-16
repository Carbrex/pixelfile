package container

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// ComputeSHA256 returns the lowercase hex-encoded SHA-256 hash of data.
func ComputeSHA256(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// VerifySHA256 compares computed SHA-256 against expected hash (case-insensitive).
func VerifySHA256(data []byte, expectedHash string) bool {
	actual := ComputeSHA256(data)
	return strings.EqualFold(actual, strings.TrimSpace(expectedHash))
}
