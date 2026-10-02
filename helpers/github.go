package helpers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"gopkg.in/yaml.v3"
)

type ghcrTokenResponse struct {
	Token string `json:"token"`
}

type GitHubRelease struct {
	TagName string `json:"tag_name"`
}

type GitHubTag struct {
	Name   string `json:"name"`
	Commit struct {
		SHA string `json:"sha"`
	} `json:"commit"`
}

type GitHubWorkflowRun struct {
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
}

type GitHubWorkflowRunsResponse struct {
	WorkflowRuns []GitHubWorkflowRun `json:"workflow_runs"`
}

type HelmChartImageTag struct {
	ImageTag string `yaml:"image_tag"`
}

func GetLatestGitHubReleaseTag(ctx context.Context, repo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "GovUKSyntheticTestApp")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("failed to get latest GitHub release: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var release GitHubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return "", err
	}

	return release.TagName, nil
}

func GetLatestGitHubTagSHA(ctx context.Context, repo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/tags", repo)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "GovUKSyntheticTestApp")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("failed to get GitHub tags: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var tags []GitHubTag
	if err := json.Unmarshal(body, &tags); err != nil {
		return "", err
	}

	if len(tags) == 0 {
		return "", fmt.Errorf("no tags found for repo %s", repo)
	}

	return tags[0].Commit.SHA, nil
}

func GetGithubWorkflowRunStatus(ctx context.Context, workflow, headSha, githubToken string) (*GitHubWorkflowRunsResponse, error) {
	var runsResp GitHubWorkflowRunsResponse
	url := fmt.Sprintf("https://api.github.com/repos/alphagov/govuk-synthetic-test-app-canary/actions/workflows/%s/runs?head_sha=%s", workflow, headSha)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+githubToken)
	req.Header.Set("X-GitHub-Api-Version", "2026-03-10")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get workflow runs: %s %s", resp.Status, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(&runsResp); err != nil {
		return nil, err
	}

	if len(runsResp.WorkflowRuns) == 0 {
		return nil, fmt.Errorf("no workflow runs found for head_sha %s", headSha)
	}

	return &runsResp, nil
}

// GetGHCRToken fetches an anonymous authentication token for GHCR using the repository scope.
func GetGHCRToken(ctx context.Context, image string) (string, error) {
	url := fmt.Sprintf("https://ghcr.io/token?service=ghcr.io&scope=repository:%s:pull", image)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "GovUKSyntheticTestApp")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to perform GET request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get GHCR token: %s %s", resp.Status, string(body))
	}

	var tokenResp ghcrTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode GHCR token response: %w", err)
	}

	return tokenResp.Token, nil
}

func GetGHCRImageDigest(ctx context.Context, repo, tag, token string) (string, error) {
	urlStr := fmt.Sprintf("https://ghcr.io/v2/%s/manifests/%s", repo, tag)

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create GET request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.oci.image.index.v1+json, application/vnd.docker.distribution.manifest.list.v2+json, application/vnd.oci.image.manifest.v1+json")
	req.Header.Set("User-Agent", "GovUKSyntheticTestApp")

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to perform GET request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get image manifest from GHCR: %s %s", resp.Status, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(body)
	return hex.EncodeToString(hash[:]), nil
}

func GetImageTagFromChartRepo(ctx context.Context, env string) (string, error) {
	url := fmt.Sprintf("http://raw.githubusercontent.com/alphagov/govuk-helm-charts/refs/heads/main/charts/app-config/image-tags/%s/govuk-synthetic-test-app-canary", env)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create GET request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to perform GET request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch URL: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var fileContent HelmChartImageTag
	err = yaml.Unmarshal(body, &fileContent)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	if fileContent.ImageTag == "" {
		return "", fmt.Errorf("image_tag not found in the YAML content")
	}

	return fileContent.ImageTag, nil
}
