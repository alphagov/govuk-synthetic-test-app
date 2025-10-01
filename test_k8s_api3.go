package main

import (
	"fmt"

	k8s_api "govuk-synthetic-test-app/helpers"
)

func main() {
	client, token, _ := k8s_api.GetK8sClient()

	k8s_api_url_all := "https://kubernetes.default.svc/api/v1/namespaces/apps/pods"
	k8s_api_url_specific_pod := "https://kubernetes.default.svc/api/v1/namespaces/apps/pods/dgu-synthetic-test-app-runner-5ddf487cc8-jfpcx"

	bodyText_all, _ := k8s_api.GetK8sAPIData(client, k8s_api_url_all, token)
	bodyText_specific, _ := k8s_api.GetK8sAPIData(client, k8s_api_url_specific_pod, token)

	fmt.Printf("%s\n", bodyText_all)
	fmt.Printf("%s\n", bodyText_specific)
}
