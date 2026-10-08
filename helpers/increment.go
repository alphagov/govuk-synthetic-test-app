package helpers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// IncrementCanaryVersion clones the specified repository, increments the version in the .version file,
// and commits the change back to the repository. It returns the new commit SHA.
func IncrementCanaryVersion(ctx context.Context, repoUrl, githubToken string) (string, error) {
	tempDir, err := os.MkdirTemp("", "canary-version-bump-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	if err := clone(ctx, repoUrl, githubToken, tempDir); err != nil {
		return "", fmt.Errorf("failed to clone repo: %w", err)
	}

	versionFilePath := filepath.Join(tempDir, ".version")

	content, err := os.ReadFile(versionFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read .version file: %w", err)
	}

	versionStr := strings.TrimSpace(string(content))
	if versionStr == "" {
		return "", fmt.Errorf(".version file is empty")
	}

	newVersion, err := incrementVersion(versionStr)
	if err != nil {
		return "", fmt.Errorf("failed to increment version: %w", err)
	}

	if err := os.WriteFile(versionFilePath, []byte(newVersion), 0o644); err != nil {
		return "", fmt.Errorf("failed to write new version: %w", err)
	}

	if err := commitAndPush(ctx, tempDir); err != nil {
		return "", fmt.Errorf("failed to commit and push: %w", err)
	}

	return getHeadSha(ctx, tempDir)
}

func incrementVersion(versionStr string) (string, error) {
	cleanVersionStr := strings.TrimPrefix(versionStr, "v")
	version, err := strconv.Atoi(cleanVersionStr)
	if err != nil {
		return "", fmt.Errorf("unsupported version format: %s", versionStr)
	}

	return fmt.Sprintf("v%d", version+1), nil
}
