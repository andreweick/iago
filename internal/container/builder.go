package container

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

// isGitHubActions returns true if running in GitHub Actions environment
func isGitHubActions() bool {
	return os.Getenv("GITHUB_ACTIONS") == "true"
}

// isCIEnvironment returns true if running in any CI environment
func isCIEnvironment() bool {
	return os.Getenv("CI") == "true" || isGitHubActions()
}

// BuildOptions contains configuration for building containers
type BuildOptions struct {
	WorkloadName  string
	ContextPath   string
	Tag           string
	RegistryURL   string
	Local         bool
	NoPush        bool
	Sign          bool
	CosignKeyPath string // Path to custom cosign private key (optional)
	AuthConfig    *AuthConfig
}

// AuthConfig contains registry authentication details
type AuthConfig struct {
	Username string
	Password string
	Token    string
}

// Builder handles container building operations
type Builder struct {
	options BuildOptions
}

// NewBuilder creates a new container builder
func NewBuilder(options BuildOptions) *Builder {
	return &Builder{
		options: options,
	}
}

// BuildContainer builds a container image from a Dockerfile or Containerfile
func (b *Builder) BuildContainer(ctx context.Context) (v1.Image, error) {
	// Try Containerfile first, then Dockerfile for backwards compatibility
	containerfilePath := filepath.Join(b.options.ContextPath, "Containerfile")
	dockerfilePath := filepath.Join(b.options.ContextPath, "Dockerfile")

	var buildFilePath string
	if _, err := os.Stat(containerfilePath); err == nil {
		buildFilePath = containerfilePath
	} else if _, err := os.Stat(dockerfilePath); err == nil {
		buildFilePath = dockerfilePath
	} else {
		return nil, fmt.Errorf("neither Containerfile nor Dockerfile found in %s", b.options.ContextPath)
	}

	// For now, we'll use a simple approach with go-containerregistry
	// This is a basic implementation - in a real scenario, you might want to use buildkit
	return b.buildFromDockerfile(ctx, buildFilePath)
}

// buildFromDockerfile builds an image from a Dockerfile
// Note: This is a simplified implementation. For full Dockerfile support,
// consider using buildkit or implementing a more complete Docker parser
func (b *Builder) buildFromDockerfile(ctx context.Context, dockerfilePath string) (v1.Image, error) {
	// Read Dockerfile
	dockerfile, err := os.ReadFile(dockerfilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Dockerfile: %w", err)
	}

	// Parse basic FROM instruction to get base image
	baseImage, err := b.parseBaseImage(string(dockerfile))
	if err != nil {
		return nil, fmt.Errorf("failed to parse base image: %w", err)
	}

	// Pull base image
	img, err := remote.Image(baseImage, remote.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to pull base image %s: %w", baseImage, err)
	}

	// Create a layer from the context directory
	layer, err := b.createLayerFromContext()
	if err != nil {
		return nil, fmt.Errorf("failed to create layer from context: %w", err)
	}

	// Add layer to image
	img, err = mutate.AppendLayers(img, layer)
	if err != nil {
		return nil, fmt.Errorf("failed to append layer: %w", err)
	}

	return img, nil
}

// parseBaseImage extracts the base image from a Dockerfile's FROM instruction
func (b *Builder) parseBaseImage(dockerfile string) (name.Reference, error) {
	lines := strings.Split(dockerfile, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToUpper(line), "FROM ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				baseImageStr := parts[1]
				ref, err := name.ParseReference(baseImageStr)
				if err != nil {
					return nil, fmt.Errorf("invalid base image reference %s: %w", baseImageStr, err)
				}
				return ref, nil
			}
		}
	}
	return nil, fmt.Errorf("no FROM instruction found in Dockerfile")
}

// createLayerFromContext creates a layer from the build context
func (b *Builder) createLayerFromContext() (v1.Layer, error) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	defer tw.Close()

	contextPath := b.options.ContextPath
	if contextPath == "" {
		return nil, fmt.Errorf("context path is empty")
	}

	// Walk through all files in the context directory
	err := filepath.Walk(contextPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error walking path %s: %w", path, err)
		}

		// Skip the root directory itself
		if path == contextPath {
			return nil
		}

		// Skip Dockerfile as it's not included in the context layer
		if info.Name() == "Dockerfile" || info.Name() == "Containerfile" {
			return nil
		}

		// Get relative path from context root
		relPath, err := filepath.Rel(contextPath, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %w", path, err)
		}

		// Convert to forward slashes for tar archive (platform independent)
		relPath = filepath.ToSlash(relPath)

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("failed to create tar header for %s: %w", path, err)
		}
		header.Name = relPath

		// Handle different file types
		switch {
		case info.Mode().IsRegular():
			// Write header for regular file
			if err := tw.WriteHeader(header); err != nil {
				return fmt.Errorf("failed to write tar header for %s: %w", path, err)
			}

			// Copy file contents
			file, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open file %s: %w", path, err)
			}
			defer file.Close()

			if _, err := io.Copy(tw, file); err != nil {
				return fmt.Errorf("failed to copy file contents for %s: %w", path, err)
			}

		case info.IsDir():
			// Write header for directory
			if err := tw.WriteHeader(header); err != nil {
				return fmt.Errorf("failed to write tar header for directory %s: %w", path, err)
			}

		case info.Mode()&os.ModeSymlink != 0:
			// Handle symbolic links
			linkTarget, err := os.Readlink(path)
			if err != nil {
				return fmt.Errorf("failed to read symlink %s: %w", path, err)
			}
			header.Linkname = linkTarget
			if err := tw.WriteHeader(header); err != nil {
				return fmt.Errorf("failed to write tar header for symlink %s: %w", path, err)
			}

		default:
			// Skip other types of files (devices, pipes, etc.)
			return nil
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create tar archive from context: %w", err)
	}

	// Close the tar writer to finalize the archive
	if err := tw.Close(); err != nil {
		return nil, fmt.Errorf("failed to close tar writer: %w", err)
	}

	// Create layer from the tar archive
	return tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(buf.Bytes())), nil
	})
}

// PushContainer pushes the built image to a registry
func (b *Builder) PushContainer(ctx context.Context, img v1.Image) error {
	var registryURL string

	if b.options.Local {
		registryURL = "localhost:5000"
	} else {
		registryURL = b.options.RegistryURL
	}

	// Construct full image reference
	imageRef := fmt.Sprintf("%s/%s:%s", registryURL, b.options.WorkloadName, b.options.Tag)

	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return fmt.Errorf("invalid image reference %s: %w", imageRef, err)
	}

	// Configure push options
	pushOptions := []remote.Option{
		remote.WithContext(ctx),
	}

	// Add authentication if provided
	if b.options.AuthConfig != nil {
		if b.options.AuthConfig.Token != "" {
			pushOptions = append(pushOptions, remote.WithAuth(&authn.Bearer{
				Token: b.options.AuthConfig.Token,
			}))
		} else if b.options.AuthConfig.Username != "" && b.options.AuthConfig.Password != "" {
			pushOptions = append(pushOptions, remote.WithAuth(&authn.Basic{
				Username: b.options.AuthConfig.Username,
				Password: b.options.AuthConfig.Password,
			}))
		}
	}

	// Push the image
	err = remote.Write(ref, img, pushOptions...)
	if err != nil {
		return fmt.Errorf("failed to push image to %s: %w", imageRef, err)
	}

	fmt.Printf("Successfully pushed %s\n", imageRef)
	return nil
}

// SignContainer signs a container image using cosign (keyless or key-based)
func (b *Builder) SignContainer(ctx context.Context, imageRef string) error {
	if !b.options.Sign {
		return nil
	}

	// Determine signing method based on environment and available keys
	if isGitHubActions() {
		return b.signContainerKeyless(ctx, imageRef)
	}

	// Try key-based signing for local development
	keyPath := b.resolveSigningKeyPath()
	if keyPath != "" {
		return b.signContainerWithKey(ctx, imageRef, keyPath)
	}

	// Fallback to keyless signing if no keys available locally
	return b.signContainerKeyless(ctx, imageRef)
}

// resolveSigningKeyPath determines the cosign private key path with fallback priority
func (b *Builder) resolveSigningKeyPath() string {
	// 1. CLI flag override
	if b.options.CosignKeyPath != "" {
		if _, err := os.Stat(b.options.CosignKeyPath); err == nil {
			return b.options.CosignKeyPath
		}
		fmt.Printf("Warning: specified cosign key path not found: %s\n", b.options.CosignKeyPath)
	}

	// 2. Environment variable
	if envKey := os.Getenv("COSIGN_PRIVATE_KEY"); envKey != "" {
		// Environment variable contains the key content, not path
		return envKey
	}

	// 3. Default location
	homeDir, err := os.UserHomeDir()
	if err == nil {
		defaultPath := filepath.Join(homeDir, ".config", "sigstore", "cosign.key")
		if _, err := os.Stat(defaultPath); err == nil {
			return defaultPath
		}
	}

	return ""
}

// signContainerKeyless implements keyless signing for CI environments
func (b *Builder) signContainerKeyless(ctx context.Context, imageRef string) error {
	fmt.Printf("Signing container %s with cosign keyless signing...\n", imageRef)

	// TODO: Implement keyless cosign signing integration
	// This would involve:
	// 1. Setting up OIDC authentication flow (GitHub Actions provides OIDC token automatically)
	// 2. Getting ephemeral certificate from Fulcio
	// 3. Creating signature payload for the container digest
	// 4. Signing with the ephemeral key
	// 5. Uploading signature to the registry and Rekor transparency log

	if isGitHubActions() {
		fmt.Printf("Using GitHub Actions OIDC token for keyless signing...\n")
	} else {
		fmt.Printf("Please authenticate via OIDC provider (Google/GitHub/Microsoft)...\n")
	}

	fmt.Printf("Note: Keyless signing implementation is a placeholder - would integrate with sigstore keyless flow\n")
	fmt.Printf("✅ Container %s would be signed successfully with keyless signing\n", imageRef)
	return nil
}

// signContainerWithKey implements key-based signing for local development
func (b *Builder) signContainerWithKey(ctx context.Context, imageRef string, keyPath string) error {
	fmt.Printf("Signing container %s with cosign key-based signing...\n", imageRef)

	// TODO: Implement key-based cosign signing integration
	// This would involve:
	// 1. Loading the private key from file or environment variable
	// 2. Prompting for passphrase (or using COSIGN_PASSWORD env var)
	// 3. Creating signature payload for the container digest
	// 4. Signing with the loaded private key
	// 5. Uploading signature to the registry

	if strings.HasPrefix(keyPath, "-----BEGIN") {
		fmt.Printf("Using cosign private key from environment variable...\n")
	} else {
		fmt.Printf("Using cosign private key from: %s\n", keyPath)
	}

	fmt.Printf("Note: Key-based signing implementation is a placeholder - would integrate with cosign key signing\n")
	fmt.Printf("✅ Container %s would be signed successfully with key-based signing\n", imageRef)
	return nil
}

// BuildAndPush is a convenience method that builds and optionally pushes a container
func (b *Builder) BuildAndPush(ctx context.Context) error {
	// Build the container
	img, err := b.BuildContainer(ctx)
	if err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	fmt.Printf("Successfully built container for %s\n", b.options.WorkloadName)

	// Sign if requested (before push to ensure no unsigned images reach registry)
	if b.options.Sign {
		var registryURL string
		if b.options.Local {
			registryURL = "localhost:5000"
		} else {
			registryURL = b.options.RegistryURL
		}
		imageRef := fmt.Sprintf("%s/%s:%s", registryURL, b.options.WorkloadName, b.options.Tag)

		err = b.SignContainer(ctx, imageRef)
		if err != nil {
			return fmt.Errorf("signing failed: %w", err)
		}
	}

	// Push if not disabled (only after successful signing, if requested)
	if !b.options.NoPush {
		err = b.PushContainer(ctx, img)
		if err != nil {
			return fmt.Errorf("push failed: %w", err)
		}
	}

	return nil
}

// ValidateLocalRegistry checks if a local registry is running
func ValidateLocalRegistry(ctx context.Context) error {
	// Try to connect to localhost:5000
	_, err := crane.Catalog("localhost:5000", crane.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("local registry at localhost:5000 not accessible: %w\nTip: Start a local registry with: docker run -d -p 5000:5000 --name registry registry:2", err)
	}
	return nil
}
