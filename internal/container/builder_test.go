package container

import (
	"archive/tar"
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBuilder(t *testing.T) {
	options := BuildOptions{
		WorkloadName: "test-workload",
		ContextPath:  "/tmp/test",
		Tag:          "latest",
		RegistryURL:  "ghcr.io/test",
		Local:        false,
		NoPush:       false,
		Sign:         false,
	}

	builder := NewBuilder(options)

	assert.NotNil(t, builder)
	assert.Equal(t, "test-workload", builder.options.WorkloadName)
	assert.Equal(t, "/tmp/test", builder.options.ContextPath)
	assert.Equal(t, "latest", builder.options.Tag)
	assert.Equal(t, "ghcr.io/test", builder.options.RegistryURL)
	assert.False(t, builder.options.Local)
	assert.False(t, builder.options.NoPush)
	assert.False(t, builder.options.Sign)
}

func TestSignContainer_SkipWhenDisabled(t *testing.T) {
	options := BuildOptions{
		WorkloadName: "test-workload",
		Sign:         false,
	}

	builder := NewBuilder(options)
	ctx := context.Background()

	err := builder.SignContainer(ctx, "test-image:latest")

	assert.NoError(t, err)
}

func TestSignContainer_PlaceholderWhenEnabled(t *testing.T) {
	options := BuildOptions{
		WorkloadName: "test-workload",
		Sign:         true,
	}

	builder := NewBuilder(options)
	ctx := context.Background()

	err := builder.SignContainer(ctx, "test-image:latest")

	assert.NoError(t, err)
}

func TestValidateLocalRegistry_NotRunning(t *testing.T) {
	ctx := context.Background()

	err := ValidateLocalRegistry(ctx)

	// Should fail since no local registry is running
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "local registry at localhost:5000 not accessible")
	assert.Contains(t, err.Error(), "docker run -d -p 5000:5000 --name registry registry:2")
}

func TestBuildOptions_AuthConfig(t *testing.T) {
	authConfig := &AuthConfig{
		Username: "testuser",
		Password: "testpass",
		Token:    "testtoken",
	}

	options := BuildOptions{
		WorkloadName: "test-workload",
		AuthConfig:   authConfig,
	}

	builder := NewBuilder(options)

	assert.NotNil(t, builder.options.AuthConfig)
	assert.Equal(t, "testuser", builder.options.AuthConfig.Username)
	assert.Equal(t, "testpass", builder.options.AuthConfig.Password)
	assert.Equal(t, "testtoken", builder.options.AuthConfig.Token)
}

func TestCreateLayerFromContext_EmptyContext(t *testing.T) {
	options := BuildOptions{
		ContextPath: "",
	}
	builder := NewBuilder(options)

	_, err := builder.createLayerFromContext()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context path is empty")
}

func TestCreateLayerFromContext_NonExistentPath(t *testing.T) {
	options := BuildOptions{
		ContextPath: "/non/existent/path",
	}
	builder := NewBuilder(options)

	_, err := builder.createLayerFromContext()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create tar archive from context")
}

func TestCreateLayerFromContext_WithFiles(t *testing.T) {
	// Create temporary context directory
	tempDir := t.TempDir()

	// Create test directory structure
	testFiles := map[string]string{
		"config/app.conf":  "key=value\nport=8080",
		"scripts/start.sh": "#!/bin/bash\necho 'Starting app'",
		"data/test.json":   `{"test": true}`,
		"README.md":        "# Test Application",
	}

	for filePath, content := range testFiles {
		fullPath := filepath.Join(tempDir, filePath)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		require.NoError(t, err)

		err = os.WriteFile(fullPath, []byte(content), 0644)
		require.NoError(t, err)
	}

	// Create Dockerfile (should be excluded from layer)
	dockerfilePath := filepath.Join(tempDir, "Dockerfile")
	err := os.WriteFile(dockerfilePath, []byte("FROM alpine"), 0644)
	require.NoError(t, err)

	// Build layer
	options := BuildOptions{
		ContextPath: tempDir,
	}
	builder := NewBuilder(options)

	layer, err := builder.createLayerFromContext()
	require.NoError(t, err)
	assert.NotNil(t, layer)

	// Verify layer contents
	rc, err := layer.Uncompressed()
	require.NoError(t, err)
	defer rc.Close()

	tr := tar.NewReader(rc)
	foundFiles := make(map[string]string)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		// Read file content if it's a regular file
		if header.Typeflag == tar.TypeReg {
			content, err := io.ReadAll(tr)
			require.NoError(t, err)
			foundFiles[header.Name] = string(content)
		} else if header.Typeflag == tar.TypeDir {
			foundFiles[header.Name] = "[DIRECTORY]"
		}
	}

	// Verify all test files are included
	assert.Equal(t, testFiles["config/app.conf"], foundFiles["config/app.conf"])
	assert.Equal(t, testFiles["scripts/start.sh"], foundFiles["scripts/start.sh"])
	assert.Equal(t, testFiles["data/test.json"], foundFiles["data/test.json"])
	assert.Equal(t, testFiles["README.md"], foundFiles["README.md"])

	// Verify directories are included
	assert.Equal(t, "[DIRECTORY]", foundFiles["config"])
	assert.Equal(t, "[DIRECTORY]", foundFiles["scripts"])
	assert.Equal(t, "[DIRECTORY]", foundFiles["data"])

	// Verify Dockerfile is NOT included
	_, dockerfileFound := foundFiles["Dockerfile"]
	assert.False(t, dockerfileFound, "Dockerfile should not be included in layer")
}

func TestCreateLayerFromContext_WithSymlinks(t *testing.T) {
	// Create temporary context directory
	tempDir := t.TempDir()

	// Create a regular file
	originalFile := filepath.Join(tempDir, "original.txt")
	err := os.WriteFile(originalFile, []byte("original content"), 0644)
	require.NoError(t, err)

	// Create a symlink to the file
	symlinkPath := filepath.Join(tempDir, "link.txt")
	err = os.Symlink("original.txt", symlinkPath)
	require.NoError(t, err)

	// Build layer
	options := BuildOptions{
		ContextPath: tempDir,
	}
	builder := NewBuilder(options)

	layer, err := builder.createLayerFromContext()
	require.NoError(t, err)
	assert.NotNil(t, layer)

	// Verify layer contents
	rc, err := layer.Uncompressed()
	require.NoError(t, err)
	defer rc.Close()

	tr := tar.NewReader(rc)
	foundSymlink := false

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		if header.Name == "link.txt" {
			assert.Equal(t, byte(tar.TypeSymlink), header.Typeflag)
			assert.Equal(t, "original.txt", header.Linkname)
			foundSymlink = true
		}
	}

	assert.True(t, foundSymlink, "Symlink should be included in layer")
}

func TestCreateLayerFromContext_WithContainerfile(t *testing.T) {
	// Create temporary context directory
	tempDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	// Create Containerfile (should be excluded from layer)
	containerfilePath := filepath.Join(tempDir, "Containerfile")
	err = os.WriteFile(containerfilePath, []byte("FROM alpine"), 0644)
	require.NoError(t, err)

	// Build layer
	options := BuildOptions{
		ContextPath: tempDir,
	}
	builder := NewBuilder(options)

	layer, err := builder.createLayerFromContext()
	require.NoError(t, err)
	assert.NotNil(t, layer)

	// Verify layer contents
	rc, err := layer.Uncompressed()
	require.NoError(t, err)
	defer rc.Close()

	tr := tar.NewReader(rc)
	foundFiles := make(map[string]bool)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		foundFiles[header.Name] = true
	}

	// Verify test file is included but Containerfile is not
	assert.True(t, foundFiles["test.txt"], "test.txt should be included")
	assert.False(t, foundFiles["Containerfile"], "Containerfile should not be included in layer")
}

func TestCreateLayerFromContext_PreservesPermissions(t *testing.T) {
	// Create temporary context directory
	tempDir := t.TempDir()

	// Create executable script
	scriptPath := filepath.Join(tempDir, "script.sh")
	err := os.WriteFile(scriptPath, []byte("#!/bin/bash\necho hello"), 0755)
	require.NoError(t, err)

	// Create regular file
	configPath := filepath.Join(tempDir, "config.txt")
	err = os.WriteFile(configPath, []byte("config=value"), 0644)
	require.NoError(t, err)

	// Build layer
	options := BuildOptions{
		ContextPath: tempDir,
	}
	builder := NewBuilder(options)

	layer, err := builder.createLayerFromContext()
	require.NoError(t, err)
	assert.NotNil(t, layer)

	// Verify layer contents and permissions
	rc, err := layer.Uncompressed()
	require.NoError(t, err)
	defer rc.Close()

	tr := tar.NewReader(rc)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		switch header.Name {
		case "script.sh":
			// Verify executable permissions are preserved
			assert.Equal(t, int64(0755), header.Mode&int64(fs.ModePerm))
		case "config.txt":
			// Verify regular file permissions are preserved
			assert.Equal(t, int64(0644), header.Mode&int64(fs.ModePerm))
		}
	}
}

func TestCreateLayerFromContext_RealWorkloadStructure(t *testing.T) {
	// Create temporary context directory that mimics real workload structure
	tempDir := t.TempDir()

	// Create structure similar to containers/caddy-work/
	workloadFiles := map[string]string{
		"config/Caddyfile":               ":80 {\n\trespond \"Hello World\"\n}",
		"scripts/health.sh":              "#!/bin/bash\ncurl -f http://localhost:80 || exit 1",
		"scripts/hello.sh":               "#!/bin/bash\necho \"Hello from iago workload\"",
		"scripts/init.sh":                "#!/bin/bash\necho \"Initializing workload...\"",
		"systemd/caddy-work-app.service": "[Unit]\nDescription=Caddy Web Server\n[Service]\nExecStart=/usr/bin/caddy run",
		"Containerfile":                  "FROM quay.io/fedora/fedora-bootc:42\nCOPY config/Caddyfile /etc/caddy/\nCOPY scripts/ /usr/local/bin/\nCOPY systemd/ /etc/systemd/system/",
	}

	for filePath, content := range workloadFiles {
		fullPath := filepath.Join(tempDir, filePath)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		require.NoError(t, err)

		// Set appropriate permissions for script files
		mode := os.FileMode(0644)
		if filepath.Ext(filePath) == ".sh" {
			mode = 0755
		}

		err = os.WriteFile(fullPath, []byte(content), mode)
		require.NoError(t, err)
	}

	// Build layer
	options := BuildOptions{
		ContextPath: tempDir,
	}
	builder := NewBuilder(options)

	layer, err := builder.createLayerFromContext()
	require.NoError(t, err, "Layer creation should succeed with real workload structure")
	assert.NotNil(t, layer)

	// Verify layer contents
	rc, err := layer.Uncompressed()
	require.NoError(t, err)
	defer rc.Close()

	tr := tar.NewReader(rc)
	foundFiles := make(map[string]string)
	foundDirs := make(map[string]bool)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		if header.Typeflag == tar.TypeReg {
			content, err := io.ReadAll(tr)
			require.NoError(t, err)
			foundFiles[header.Name] = string(content)

			// Verify script permissions are preserved
			if filepath.Ext(header.Name) == ".sh" {
				assert.Equal(t, int64(0755), header.Mode&int64(fs.ModePerm),
					"Script %s should have executable permissions", header.Name)
			}
		} else if header.Typeflag == tar.TypeDir {
			foundDirs[header.Name] = true
		}
	}

	// Verify all expected files are included (except Containerfile)
	expectedFiles := []string{
		"config/Caddyfile",
		"scripts/health.sh",
		"scripts/hello.sh",
		"scripts/init.sh",
		"systemd/caddy-work-app.service",
	}

	for _, expectedFile := range expectedFiles {
		_, found := foundFiles[expectedFile]
		assert.True(t, found, "Expected file %s should be in layer", expectedFile)
	}

	// Verify directories are created
	expectedDirs := []string{"config", "scripts", "systemd"}
	for _, expectedDir := range expectedDirs {
		assert.True(t, foundDirs[expectedDir], "Expected directory %s should be in layer", expectedDir)
	}

	// Verify Containerfile is NOT included in layer
	_, containerfileFound := foundFiles["Containerfile"]
	assert.False(t, containerfileFound, "Containerfile should not be included in layer")

	// Verify specific file contents
	assert.Contains(t, foundFiles["config/Caddyfile"], "Hello World",
		"Caddyfile should contain expected content")
	assert.Contains(t, foundFiles["scripts/health.sh"], "curl -f",
		"Health script should contain expected content")
	assert.Contains(t, foundFiles["systemd/caddy-work-app.service"], "Description=Caddy",
		"Systemd service should contain expected content")
}
