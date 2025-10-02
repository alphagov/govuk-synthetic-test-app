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

func GetK8sAPIData(client *http.Client, token string, namespace string, resource_type string) ([]byte, error) {
	req, err := http.NewRequest("GET", API_SERVER+"/"+namespace+"/"+resource_type, nil)
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

func GetPodList(namespace string, kind string) (*corev1.PodList, error) {
	client, token, _ := GetK8sClient()

	bodyText_all, _ := GetK8sAPIData(client, token, namespace, kind)

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
