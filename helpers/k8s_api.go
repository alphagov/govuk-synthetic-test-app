package helpers

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"os"

	"sigs.k8s.io/aws-iam-authenticator/pkg/token"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

const API_SERVER string = "https://kubernetes.default.svc/api/v1/namespaces"
const CERT_PATH string = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
const INTEGRATION_AWS_ACCOUNT_ID string = "210287912431"
const STAGING_AWS_ACCOUNT_ID string = "696911096973"
const PRODUCTION_AWS_ACCOUNT_ID string = "172025368201"

func checkRunningInK8s() (bool, error) {
	if _, err := os.Stat(CERT_PATH); err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("Can only run this code when inside a k8s pod\n")
			return false, nil
		} else {
			return false, err
		}
	}
	return true, nil
}

func GetK8sClient(environment_account_id string) (*http.Client, string, error) {
	ctx := context.TODO()

	running_in_k8s, err := checkRunningInK8s()
	if err != nil {
		return nil, "", err
	} else if !running_in_k8s {
		return nil, "", nil
	}

	g, _ := token.NewGenerator(false, false)
	tk, err := g.GetWithOptions(ctx, &token.GetTokenOptions{
		Region:        "eu-west-1",
		ClusterID:     "govuk",
		AssumeRoleARN: fmt.Sprintf("arn:aws:iam::%+v:role/synthetic-test-assumed", environment_account_id),
		SessionName:   "GovUKSyntheticTestApp",
	})
	fmt.Printf("token: %+v\n", tk)
	if err != nil {
		return nil, "", err
	}

	caCert, err := os.ReadFile(CERT_PATH)
	if err != nil {
		return nil, tk.Token, err
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
		return nil, tk.Token, err
	}

	return client, tk.Token, err
}

func GetK8sAPIData(environment_account_id string, namespace string, resource_type string) ([]byte, error) {
	client, token, err := GetK8sClient(environment_account_id)
	if err != nil {
		return nil, err
	}

	url := API_SERVER + "/" + namespace + "/" + resource_type
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
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

func GetPodList(environment_account_id string, namespace string) (*corev1.PodList, error) {
	bodyText_all, err := GetK8sAPIData(environment_account_id, namespace, "pods")
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
		return nil, err
	}
	podList := podObject.(*corev1.PodList)
	return podList, nil
}
