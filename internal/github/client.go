package github

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// FetchSSHKeys fetches public SSH keys for a GitHub user
func FetchSSHKeys(username string) ([]string, error) {
	if username == "" {
		return nil, fmt.Errorf("github username cannot be empty")
	}

	url := fmt.Sprintf("https://github.com/%s.keys", username)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch SSH keys from GitHub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("GitHub user '%s' not found", username)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch SSH keys: HTTP %d", resp.StatusCode)
	}

	// Read and parse the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Split by newlines and filter out empty lines
	scanner := bufio.NewScanner(strings.NewReader(string(body)))
	var keys []string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			keys = append(keys, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse SSH keys: %w", err)
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf("no SSH keys found for GitHub user '%s'", username)
	}

	return keys, nil
}
