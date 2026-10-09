package helpers

import (
	"bytes"
	"context"
	"net/http"
)

func TriggerArgoSync(ctx context.Context, client *K8sClient) (int, error) {
	payload := []byte(`{"operation": {"initiatedBy": {"username": "synthetic-test-app"}, "sync": {"revision": "HEAD"}}}`)

	url := "/apis/argoproj.io/v1alpha1/namespaces/cluster-services/applications/govuk-synthetic-test-app-canary"

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPatch, url, bytes.NewBuffer(payload))
	if err != nil {
		return -1, err
	}

	req.Header.Set("Content-Type", "application/merge-patch+json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Client.Do(req)
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}
