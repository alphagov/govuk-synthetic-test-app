package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"sigs.k8s.io/aws-iam-authenticator/pkg/token"
)

func main() {
	g, _ := token.NewGenerator(false, false)
	tk, err := g.GetWithOptions(&token.GetTokenOptions{
		Region:        "<region>",
		ClusterID:     "<cluster_name>",
		AssumeRoleARN: "arn:aws:iam::210287912431:role/synthetic-test-assumed",
		Session:       nil,
	})

	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(tk.Token)

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

	req, err := http.NewRequest("GET", "https://kubernetes.default.svc/api/v1/namespaces/default/pods/deploy/dgu-synthetic-test-app-runner", nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+tk.Token)
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", bodyText)
}
