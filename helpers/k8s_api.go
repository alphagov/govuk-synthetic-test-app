package helpers

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"log"
	"net/http"
	"os"

	"sigs.k8s.io/aws-iam-authenticator/pkg/token"
)

func GetK8sClient() (*http.Client, string, error) {
	ctx := context.TODO()
	g, _ := token.NewGenerator(false, false)
	tk, err := g.GetWithOptions(ctx, &token.GetTokenOptions{
		Region:        "eu-west-1",
		ClusterID:     "govuk",
		AssumeRoleARN: "arn:aws:iam::210287912431:role/synthetic-test-assumed",
		SessionName:   "GovUKSyntheticTestApp",
	})
	if err != nil {
		log.Fatal(err)
	}

	caCert, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
	if err != nil {
		log.Fatal(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		},
	}
	if err != nil {
		log.Fatal(err)
	}

	return client, tk.Token, nil
}

func GetK8sAPIData(client *http.Client, k8s_api_url string, token string) ([]byte, error) {
	req, err := http.NewRequest("GET", k8s_api_url, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/yaml")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	return bodyText, nil
}
