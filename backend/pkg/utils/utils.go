package utils

import (
	"crypto/md5"
	"fmt"
	"time"
)

// GenerateID generates a unique ID
func GenerateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// MD5 calculates MD5 hash
func MD5(data string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(data)))
}

// Contains checks if a slice contains an element
func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Map converts a slice to a map
func Map(slice []string) map[string]bool {
	result := make(map[string]bool)
	for _, item := range slice {
		result[item] = true
	}
	return result
}
