package machine

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateMAC(t *testing.T) {
	prefix := "02:05:56"
	mac, err := GenerateMAC(prefix)

	assert.NoError(t, err)
	assert.NotEmpty(t, mac)
	assert.True(t, strings.HasPrefix(mac, prefix), "MAC should start with prefix")
	assert.Equal(t, 17, len(mac), "MAC should be 17 characters long")

	// Generate another MAC and ensure they're different
	mac2, err := GenerateMAC(prefix)
	assert.NoError(t, err)
	assert.NotEqual(t, mac, mac2, "Generated MACs should be unique")
}

func TestValidateMAC(t *testing.T) {
	tests := []struct {
		name     string
		mac      string
		expected bool
	}{
		{"valid MAC", "02:05:56:ab:cd:ef", true},
		{"invalid - too short", "02:05:56:ab:cd", false},
		{"invalid - too long", "02:05:56:ab:cd:ef:12", false},
		{"invalid - wrong format", "02-05-56-ab-cd-ef", false},
		{"valid - uppercase", "02:05:56:AB:CD:EF", true},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateMAC(tt.mac)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetMACOrGenerate(t *testing.T) {
	prefix := "02:05:56"

	// Test with valid existing MAC
	existingMAC := "02:05:56:aa:bb:cc"
	result, err := GetMACOrGenerate(existingMAC, prefix)
	assert.NoError(t, err)
	assert.Equal(t, existingMAC, result)

	// Test with invalid existing MAC - should generate new
	invalidMAC := "invalid-mac"
	result, err = GetMACOrGenerate(invalidMAC, prefix)
	assert.NoError(t, err)
	assert.NotEqual(t, invalidMAC, result)
	assert.True(t, strings.HasPrefix(result, prefix))

	// Test with empty MAC - should generate new
	result, err = GetMACOrGenerate("", prefix)
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(result, prefix))
	assert.True(t, ValidateMAC(result))
}
