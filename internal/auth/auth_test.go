package auth

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetAuthConfig_Priority(t *testing.T) {
	ctx := context.Background()

	// Test CLI token takes precedence
	config, err := GetAuthConfig(ctx, "testuser", "cli-token")
	assert.NoError(t, err)
	assert.Equal(t, "cli-token", config.Token)
	assert.Equal(t, "testuser", config.Username)
	assert.Equal(t, "cli", config.Source)

	// Test environment variable fallback
	os.Setenv("GITHUB_TOKEN", "env-token")
	defer os.Unsetenv("GITHUB_TOKEN")

	config, err = GetAuthConfig(ctx, "testuser", "")
	assert.NoError(t, err)
	assert.Equal(t, "env-token", config.Token)
	assert.Equal(t, "testuser", config.Username)
	assert.Equal(t, "env", config.Source)

	// Test no auth available - need to unset both GITHUB_TOKEN and OP_SERVICE_ACCOUNT_TOKEN
	os.Unsetenv("GITHUB_TOKEN")
	originalOpToken := os.Getenv("OP_SERVICE_ACCOUNT_TOKEN")
	os.Unsetenv("OP_SERVICE_ACCOUNT_TOKEN")
	defer func() {
		if originalOpToken != "" {
			os.Setenv("OP_SERVICE_ACCOUNT_TOKEN", originalOpToken)
		}
	}()

	config, err = GetAuthConfig(ctx, "", "")
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "no authentication available")
}

func TestAuthConfig_ToContainerAuthConfig(t *testing.T) {
	authConfig := &AuthConfig{
		Username: "testuser",
		Token:    "testtoken",
		Source:   "test",
	}

	containerAuth := authConfig.ToContainerAuthConfig()
	assert.Equal(t, "testuser", containerAuth.Username)
	assert.Equal(t, "testtoken", containerAuth.Password)
	assert.Equal(t, "testtoken", containerAuth.Token)

	// Test nil case
	var nilAuth *AuthConfig
	assert.Nil(t, nilAuth.ToContainerAuthConfig())
}
