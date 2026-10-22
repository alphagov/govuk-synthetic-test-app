package helpers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// IncrementCanaryVersion clones the canary repo, increments the version in the .version file,
// and commits the change back to the repo. It returns the new commit SHA.
func IncrementCanaryVersion(ctx context.Context, repoUrl, beforeCommitTag, githubToken string) (string, error) {
	tempDir, err := os.MkdirTemp(".", "canary-version-bump-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	defer func() {
		closeErr := os.RemoveAll(tempDir)

		if err == nil {
			err = closeErr
		}
	}()

	tokenUrl, err := addTokenToUrl(repoUrl, githubToken)
	if err != nil {
		return "", fmt.Errorf("failed to add token to url: %w", err)
	}

	if err := clone(ctx, tokenUrl, tempDir); err != nil {
		return "", fmt.Errorf("failed to clone repo: %w", err)
	}

	versionFilePath := filepath.Join(tempDir, ".version")

	versionStr := strings.TrimSpace(string(beforeCommitTag))
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

	if err := commitAndPush(ctx, tempDir, tokenUrl, newVersion); err != nil {
		return "", fmt.Errorf("failed to commit and push: %w", err)
	}

	headSha, err := getHeadSha(ctx, tempDir)
	if err != nil {
		return "", err
	}

	return headSha, err
}

func incrementVersion(versionStr string) (string, error) {
	cleanVersionStr := strings.TrimPrefix(versionStr, "v")
	version, err := strconv.Atoi(cleanVersionStr)
	if err != nil {
		return "", fmt.Errorf("unsupported version format: %s", versionStr)
	}

	return fmt.Sprintf("v%d", version+1), nil
}
