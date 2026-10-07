package helpers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"gopkg.in/yaml.v3"
)

type HelmChartImageTag struct {
	ImageTag string `yaml:"image_tag"`
}

type ArgoWorkflowStatus struct {
	Items []struct {
		Status struct {
			Phase string `json:"phase"`
		} `json:"status"`
	} `json:"items"`
}

type ArgoCDApplicationStatus struct {
	Status struct {
		Sync struct {
			Status string `json:"status"`
		} `json:"sync"`
		Health struct {
			Status string `json:"status"`
		} `json:"health"`
	} `json:"status"`
}

type PodList struct {
	Items []struct {
		Status struct {
			ContainerStatuses []struct {
				Name    string `json:"name"`
				Ready   bool   `json:"ready"`
				Started bool   `json:"started"`
				Image   string `json:"image"`
				ImageId string `json:"imageID"`
			} `json:"containerStatuses"`
		} `json:"status"`
		Spec struct {
			Containers []struct {
				Image string `json:"image"`
			} `json:"containers"`
		} `json:"spec"`
	} `json:"items"`
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

func GetArgoWorkflowStatus(ctx context.Context, k8sClient *K8sClient, namespace, labels string) (bool, error) {
	encodedLabels := url.QueryEscape(labels)
	url := fmt.Sprintf("/apis/argoproj.io/v1alpha1/namespaces/%s/workflows?labelSelector=%s", namespace, encodedLabels)
	resp, err := k8sClient.Get(url)
	if err != nil {
		fmt.Printf("Error reading resp: %v", resp)
		return false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading body: %s", body)
		return false, err
	}

	if resp.StatusCode != 200 {
		return false, fmt.Errorf("failed to get ArgoCD application: %s %s", resp.Status, body)
	}

	var workflow ArgoWorkflowStatus
	if err := json.Unmarshal(body, &workflow); err != nil {
		return false, err
	}

	return workflow.Items[0].Status.Phase == "Succeeded", nil
}

func GetArgoCDApplicationStatus(ctx context.Context, k8sClient *K8sClient, namespace, appName string) (bool, error) {
	url := fmt.Sprintf("/apis/argoproj.io/v1alpha1/namespaces/%s/applications/%s", namespace, appName)
	resp, err := k8sClient.Get(url)
	if err != nil {
		fmt.Printf("Error reading resp: %v", resp)
		return false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading body: %s", body)
		return false, err
	}

	if resp.StatusCode != 200 {
		return false, fmt.Errorf("failed to get ArgoCD application: %s %s", resp.Status, body)
	}

	var app ArgoCDApplicationStatus
	if err := json.Unmarshal(body, &app); err != nil {
		return false, err
	}

	return app.Status.Sync.Status == "Synced" && app.Status.Health.Status == "Healthy", nil
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

type ghcrTokenResponse struct {
	Token string `json:"token"`
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
		return "", err
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
		return "", err
	}

	req.Header.Set("Accept", "application/vnd.oci.image.index.v1+json, application/vnd.docker.distribution.manifest.list.v2+json, application/vnd.oci.image.manifest.v1+json")
	req.Header.Set("User-Agent", "GovUKSyntheticTestApp")

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
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

func GetPodImageDetails(ctx context.Context, k8sClient *K8sClient, namespace, appLabelSelector string) (tag string, sha string, err error) {
	url := fmt.Sprintf("/api/v1/namespaces/%s/pods?labelSelector=%s", namespace, appLabelSelector)
	resp, err := k8sClient.Get(url)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", "", fmt.Errorf("failed to get pods: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	var podList PodList

	if err := json.Unmarshal(body, &podList); err != nil {
		return "", "", err
	}

	if len(podList.Items) == 0 {
		return "", "", fmt.Errorf("no pods found for selector %s", appLabelSelector)
	}

	itemIdx, statusIdx := getAppPodIndexes(podList)
	if itemIdx == -1 && statusIdx == -1 {
		return "", "", fmt.Errorf("cannot find running imgae for 'app'")
	}

	image := podList.Items[itemIdx].Status.ContainerStatuses[statusIdx].Image
	imageId := podList.Items[itemIdx].Status.ContainerStatuses[statusIdx].ImageId

	return strings.Split(image, ":")[1], strings.TrimPrefix(strings.Split(imageId, "@")[1], "sha256:"), nil
}

func GetImageTagFromChartRepo(env string) (string, error) {
	url := fmt.Sprintf("http://raw.githubusercontent.com/alphagov/govuk-helm-charts/refs/heads/main/charts/app-config/image-tags/%s/govuk-synthetic-test-app-canary", env)
	resp, err := http.Get(url)
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

func getAppPodIndexes(podList PodList) (int, int) {
	for pIdx, p := range podList.Items {
		for cIdx, c := range p.Status.ContainerStatuses {
			if c.Name == "app" && c.Ready && c.Started {
				return pIdx, cIdx
			}
		}
	}
	return -1, -1
}
