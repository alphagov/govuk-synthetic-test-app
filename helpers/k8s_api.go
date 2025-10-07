package helpers

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"sigs.k8s.io/aws-iam-authenticator/pkg/token"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

var API_SERVER string = "https://kubernetes.default.svc/api/v1/namespaces"

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

func GetK8sAPIData(namespace string, resource_type string) ([]byte, error) {
	client, token, _ := GetK8sClient()
	url := API_SERVER + "/" + namespace + "/" + resource_type
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/yaml")

	resp, err := client.Do(req)
	if err != nil {
		err = fmt.Errorf("Error got %v status, retrieving %v", resp.StatusCode, url)
	}
	defer resp.Body.Close()
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("Error got %v status, retrieving %v", resp.StatusCode, url)
	}

	if resp.StatusCode != 200 {
		err = fmt.Errorf("Error got %v status, retrieving %v", resp.StatusCode, url)
	}

	return bodyText, err
}

func GetPodList(namespace string, kind string) (*corev1.PodList, error) {
	bodyText_all, err := GetK8sAPIData(namespace, kind)
	if err != nil {
		return nil, err
	}

	// https://godoc.org/k8s.io/apimachinery/pkg/runtime#Scheme
	scheme := runtime.NewScheme()

	// https://godoc.org/k8s.io/apimachinery/pkg/runtime/serializer#CodecFactory
	codecFactory := serializer.NewCodecFactory(scheme)

	// https://godoc.org/k8s.io/apimachinery/pkg/runtime#Decoder
	deserializer := codecFactory.UniversalDeserializer()

	podObject, _, err := deserializer.Decode(bodyText_all, nil, &corev1.PodList{})
	if err != nil {
		log.Fatal(err)
	}
	podList := podObject.(*corev1.PodList)
	return podList, nil
}
