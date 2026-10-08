package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

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

func GetVersionFromApp(ctx context.Context) (string, error) {
	resp, err := http.Get("http://govuk-synthetic-test-app-canary.apps.svc.cluster.local")
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
