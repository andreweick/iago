package machine

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateRandomPassword(t *testing.T) {
	password, err := GenerateRandomPassword()

	assert.NoError(t, err)
	assert.NotEmpty(t, password)
	assert.True(t, len(password) > 10, "Password should be sufficiently long")

	// Generate another password and ensure they're different
	password2, err := GenerateRandomPassword()
	assert.NoError(t, err)
	assert.NotEqual(t, password, password2, "Generated passwords should be unique")
}

func TestGeneratePasswordHash(t *testing.T) {
	password := "test-password"
	hash, err := GeneratePasswordHash(password)

	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash, "Hash should be different from password")
	assert.True(t, len(hash) > 20, "Hash should be sufficiently long")
}
