// backend/pkg/utils/session.go
package utils

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// GenerateSessionID generates a session ID based on input string
func GenerateSessionID(input string) string {
	// Create a hash of the input combined with timestamp
	hash := md5.Sum([]byte(input + fmt.Sprintf("%d", time.Now().Unix()/3600))) // Changes every hour
	return hex.EncodeToString(hash[:])[:16] // Return first 16 characters
}

// MD5Hash generates MD5 hash of input string
func MD5Hash(input string) string {
	hash := md5.Sum([]byte(input))
	return hex.EncodeToString(hash[:])
}

// GenerateRandomID generates a random ID
func GenerateRandomID(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)[:length]
}

// ValidateSessionID validates if a session ID format is correct
func ValidateSessionID(sessionID string) bool {
	if len(sessionID) != 16 {
		return false
	}
	
	// Check if it's a valid hex string
	_, err := hex.DecodeString(sessionID)
	return err == nil
}