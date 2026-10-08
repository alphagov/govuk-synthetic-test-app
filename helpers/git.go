package helpers

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"strings"
)

func clone(ctx context.Context, repoUrl, githubToken, tempDir string) error {
	u, err := url.Parse(repoUrl)
	if err == nil && u.Host != "" && (strings.HasSuffix(u.Host, ".github.com") || u.Host == "github.com") {
		u.User = url.UserPassword("x-access-token", githubToken)
		repoUrl = u.String()
	}

	cmd := exec.CommandContext(ctx, "git", "clone", repoUrl, tempDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to clone repo: %w, output: %s", err, string(output))
	}

	return nil
}

func commitAndPush(ctx context.Context, repoDir string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", repoDir, "add", ".version")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to git add: %w, output: %s", err, string(output))
	}

	cmd = exec.CommandContext(ctx, "git", "-C", repoDir, "commit", "-m", "Bump version")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to git commit: %w, output: %s", err, string(output))
	}

	cmd = exec.CommandContext(ctx, "git", "-C", repoDir, "push")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to git push: %w, output: %s", err, string(output))
	}

	return nil
}

func getHeadSha(ctx context.Context, repoDir string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", repoDir, "rev-parse", "HEAD")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get commit SHA: %w, output: %s", err, string(output))
	}

	return strings.TrimSpace(string(output)), nil
}
