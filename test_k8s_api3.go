package main

import (
	"bytes"
	"fmt"
	"log"

	k8s_api "govuk-synthetic-test-app/helpers"

	appsv1 "k8s.io/api/apps/v1"
	k8Yaml "k8s.io/apimachinery/pkg/util/yaml"
)

func main() {
	client, token, _ := k8s_api.GetK8sClient()

	k8s_api_url_all := "https://kubernetes.default.svc/api/v1/namespaces/apps/pods"
	k8s_api_url_specific_pod := "https://kubernetes.default.svc/api/v1/namespaces/apps/pods/dgu-synthetic-test-app-runner-5669b9c4cc-9tc7c"

	bodyText_all, _ := k8s_api.GetK8sAPIData(client, k8s_api_url_all, token)
	bodyText_specific, _ := k8s_api.GetK8sAPIData(client, k8s_api_url_specific_pod, token)

	fmt.Printf("%s\n", bodyText_all)
	fmt.Printf("%s\n", bodyText_specific)

	d := &appsv1.Deployment{}
	dec := k8Yaml.NewYAMLOrJSONDecoder(bytes.NewReader(bodyText_all), 1000)

	if err := dec.Decode(&d); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%v\n", d)
}
