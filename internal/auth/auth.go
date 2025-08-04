package auth

import (
	"context"
	"fmt"
	"os"

	"github.com/1password/onepassword-sdk-go"
	"github.com/andreweick/iago/internal/container"
)

// DefaultOnePasswordSecretRef is the default 1Password secret reference for GitHub tokens
const DefaultOnePasswordSecretRef = "op://iago/yq55ghqtbgmwvxbnc2xpzix4xm/credential"

// AuthConfig contains registry authentication configuration
type AuthConfig struct {
	Username string
	Token    string
	Source   string // For debugging: "cli", "env", "1password"
}

// GetAuthConfig resolves authentication using priority chain:
// 1. CLI flags (highest priority)
// 2. Environment variables
// 3. 1Password (if OP_SERVICE_ACCOUNT_TOKEN is set)
func GetAuthConfig(ctx context.Context, cliUsername, cliToken string) (*AuthConfig, error) {
	// Priority 1: CLI flags
	if cliToken != "" {
		return &AuthConfig{
			Username: cliUsername,
			Token:    cliToken,
			Source:   "cli",
		}, nil
	}

	// Priority 2: Environment variables
	if githubToken := os.Getenv("GITHUB_TOKEN"); githubToken != "" {
		return &AuthConfig{
			Username: cliUsername, // Use CLI username if provided, empty otherwise
			Token:    githubToken,
			Source:   "env",
		}, nil
	}

	// Priority 3: 1Password
	if opToken := os.Getenv("OP_SERVICE_ACCOUNT_TOKEN"); opToken != "" {
		return getAuthFrom1Password(ctx, cliUsername, opToken)
	}

	// No authentication available
	return nil, fmt.Errorf("no authentication available: set --token flag, GITHUB_TOKEN env var, or OP_SERVICE_ACCOUNT_TOKEN for 1Password integration")
}

// getAuthFrom1Password retrieves GitHub token from 1Password using the SDK
func getAuthFrom1Password(ctx context.Context, username, serviceAccountToken string) (*AuthConfig, error) {
	// Create 1Password client
	client, err := onepassword.NewClient(
		ctx,
		onepassword.WithServiceAccountToken(serviceAccountToken),
		onepassword.WithIntegrationInfo("Iago Container Registry Auth", "v1.0.0"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create 1Password client: %w", err)
	}

	// Use the default 1Password secret reference - change DefaultOnePasswordSecretRef constant to customize
	secretRef := DefaultOnePasswordSecretRef

	// Resolve the secret
	token, err := client.Secrets().Resolve(ctx, secretRef)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve 1Password secret '%s': %w", secretRef, err)
	}

	return &AuthConfig{
		Username: username, // Use CLI username if provided, empty otherwise
		Token:    token,
		Source:   "1password",
	}, nil
}

// ToContainerAuthConfig converts to the container package's AuthConfig format
func (ac *AuthConfig) ToContainerAuthConfig() *container.AuthConfig {
	if ac == nil {
		return nil
	}
	return &container.AuthConfig{
		Username: ac.Username,
		Password: ac.Token, // GitHub uses token in password field
		Token:    ac.Token,
	}
}
