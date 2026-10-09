package helpers

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"strings"
)

func ConfigureGit(ctx context.Context) error {
	botId := 340069771
	botName := "gov-uk-synthetic-test-app-canary[bot]"
	botEmail := fmt.Sprintf("%d+%s@users.noreply.github.com", botId, botName)

	if err := exec.CommandContext(ctx, "git", "config", "--global", "user.name", botName).Run(); err != nil {
		return fmt.Errorf("failed to set git user.name: %w", err)
	}
	if err := exec.CommandContext(ctx, "git", "config", "--global", "user.email", botEmail).Run(); err != nil {
		return fmt.Errorf("failed to set git user.email: %w", err)
	}
	return nil
}

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

func commitAndPush(ctx context.Context, repoDir, repoUrl, githubToken, sourceBranch string) error {
	targetBranch := "main"

	u, err := url.Parse(repoUrl)
	if err == nil && u.Host != "" && (strings.HasSuffix(u.Host, ".github.com") || u.Host == "github.com") {
		u.User = url.UserPassword("x-access-token", githubToken)
		repoUrl = u.String()
	}

	cmd := exec.CommandContext(ctx, "git", "-C", repoDir, "checkout", "-b", sourceBranch)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to git checkout: %w, output: %s", err, string(output))
	}

	cmd = exec.CommandContext(ctx, "git", "-C", repoDir, "add", ".version")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to git add: %w, output: %s", err, string(output))
	}

	cmd = exec.CommandContext(ctx, "git", "-C", repoDir, "commit", "-m", "Bump version: "+sourceBranch)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to git commit: %w, output: %s", err, string(output))
	}

	if err := mergeBranch(ctx, repoDir, sourceBranch, targetBranch); err != nil {
		return fmt.Errorf("failed to merge git commit: %s", err)
	}

	cmd = exec.CommandContext(ctx, "git", "-C", repoDir, "push", repoUrl)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to push %s: %w, output: %s", targetBranch, err, string(output))
	}
	return nil
}

func getHeadSha(ctx context.Context, repoDir string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", repoDir, "rev-parse", "HEAD")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get commit SHA: %w, output: %s", err, string(output))
	}

	return string(output), nil
}

// MergeBranch merges sourceBranch into targetBranch using --no-ff to ensure a merge commit is created,
// then pushes the targetBranch to origin.
func mergeBranch(ctx context.Context, repoDir, sourceBranch, targetBranch string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", repoDir, "checkout", targetBranch)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to checkout %s: %w, output: %s", targetBranch, err, string(output))
	}

	// Merge source branch into target branch using --no-ff
	// This forces the creation of a merge commit even if the merge could be fast-forwarded.
	mergeMsg := fmt.Sprintf("Merge branch '%s' into %s", sourceBranch, targetBranch)
	cmd = exec.CommandContext(ctx, "git", "-C", repoDir, "merge", "--no-ff", sourceBranch, "-m", mergeMsg)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to merge %s into %s: %w, output: %s", sourceBranch, targetBranch, err, string(output))
	}

	return nil
}
