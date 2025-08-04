package machine

import (
	"crypto/rand"
	"fmt"
	"strings"
)

// DefaultMACPrefix is the default MAC prefix for generated MAC addresses
const DefaultMACPrefix = "02:05:56"

// GenerateMAC generates a random MAC address with the given prefix
func GenerateMAC(prefix string) (string, error) {
	// Generate 3 random bytes for the last 3 octets
	bytes := make([]byte, 3)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Format the MAC address
	return fmt.Sprintf("%s:%02x:%02x:%02x", prefix, bytes[0], bytes[1], bytes[2]), nil
}

// ValidateMAC validates that a MAC address follows the expected format
func ValidateMAC(mac string) bool {
	parts := strings.Split(mac, ":")
	if len(parts) != 6 {
		return false
	}

	for _, part := range parts {
		if len(part) != 2 {
			return false
		}
	}

	return true
}

// GetMACOrGenerate returns the existing MAC if valid, or generates a new one
func GetMACOrGenerate(existingMAC, prefix string) (string, error) {
	if existingMAC != "" && ValidateMAC(existingMAC) {
		return existingMAC, nil
	}

	return GenerateMAC(prefix)
}
