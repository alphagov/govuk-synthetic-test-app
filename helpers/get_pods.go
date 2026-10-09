package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"gopkg.in/yaml.v3"
)

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

func GetPodImageDetails(ctx context.Context, k8sClient *K8sClient, namespace, appLabelSelector string) (tag string, sha string, err error) {
	url := fmt.Sprintf("/api/v1/namespaces/%s/pods?labelSelector=%s", namespace, appLabelSelector)
	resp, err := k8sClient.Get(ctx, url)
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

func GetVersionFromApp(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "http://govuk-synthetic-test-app-canary.apps.svc.cluster.local", nil)
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

	var pageContent string
	err = yaml.Unmarshal(body, &pageContent)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	if pageContent == "" {
		return "", fmt.Errorf("image_tag not found in the YAML content")
	}

	return pageContent, nil
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
