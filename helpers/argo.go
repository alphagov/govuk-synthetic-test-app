package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
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

	if len(workflow.Items) == 0 {
		return false, fmt.Errorf("There are no workflow items")
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
