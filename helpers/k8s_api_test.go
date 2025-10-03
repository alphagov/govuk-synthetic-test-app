package helpers_test

import (
	"fmt"
	k8s_api "govuk-synthetic-test-app/helpers"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AssumeRole", Ordered, func() {
	var client *http.Client
	var token string

	BeforeAll(func() {
		client, token, _ := k8s_api.GetK8sClient()
		Expect(client).NotTo(BeNil())
		Expect(token).NotTo(BeNil())
	})

	Context("when called with apps namespace and pods kind", func() {
		It("returns pods list with first item arch as arm64", func() {
			podList, _ := k8s_api.GetPodList(client, token, "apps", "pods")
			fmt.Printf("Pods: %+v, %+v\n", podList.Items[0].Labels["app"], podList.Items[0].Spec.Containers[0].Image)
			Expect(podList.Items[0].Labels["app.kubernetes.io/arch"]).To(Equal("arm64"))
			var allARM64 = true
			for _, item := range podList.Items {
				if val, ok := item.Labels["app.kubernetes.io/arch"]; ok {
					if val != "arm64" {
						fmt.Printf("Found non-arm64 pod: %+v\n", item.Labels["app"])
						allARM64 = false
					}
				}
			}
			Expect(allARM64).To(Equal(true))
		})
	})
})
