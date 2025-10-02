package main

import (
	"fmt"

	k8s_api "govuk-synthetic-test-app/helpers"
)

func main() {
	podList, _ := k8s_api.GetPodList("apps", "pods")

	fmt.Println("==== podList ====")
	fmt.Printf("Pods: %+v, %+v\n", podList.Items[0].Labels["app"], podList.Items[0].Spec.Containers[0].Image)
}
