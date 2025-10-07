package helpers_test

import (
	"fmt"
	k8s_api "govuk-synthetic-test-app/helpers"
	"log"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Synthetic Test Assumed role", func() {
	Context("when calling k8s api with apps namespace and pods kind", func() {
		It("returns pods list and can access the first image value", func() {
			podList, _ := k8s_api.GetPodList("apps", "pods")
			fmt.Printf("First pod image: %+v, %+v\n", podList.Items[0].Labels["app"], podList.Items[0].Spec.Containers[0].Image)
			Expect(podList.Items[0].Spec.Containers[0].Image).NotTo(BeNil())
		})
		It("returns pods list and all pods are running with arch arm64", func() {
			podList, _ := k8s_api.GetPodList("apps", "pods")
			Expect(podList.Items[0].Labels["app.kubernetes.io/arch"]).To(Equal("arm64"))
			var allARM64 = true
			for _, item := range podList.Items {
				if val, ok := item.Labels["app.kubernetes.io/arch"]; ok {
					if val != "arm64" {
						fmt.Printf("Found non-arm64 pod: %+v\n", item.Labels["app"])
						allARM64 = false
					}
				} else {
					fmt.Printf("Found pod missing arch label: %+v\n", item.Name)
				}
			}
			Expect(allARM64).To(Equal(true))
		})
	})

	Context("when calling k8s api with NON 'apps' namespace and `pods` kind", func() {
		DescribeTable("assumed role cannot access the namespace",
			func(namespace string) {
				podList, err := k8s_api.GetPodList(namespace, "pods")
				Expect(podList).To(BeNil())
				expected_string := fmt.Sprintf("Error got 403 status, retrieving https://kubernetes.default.svc/api/v1/namespaces/%+v/pods", namespace)
				Expect(err.Error()).To(Equal(expected_string))
			},
			Entry("for datagovuk", "datagovuk"),
			Entry("for default", "default"),
			Entry("for cluster-services", "cluster-services"),
			Entry("for licensify", "licensify"),
			Entry("for monitoring", "monitoring"),
		)
	})

	Context("when trying to perform a DELETE, PATCH, POST, PUT with the k8s api on the apps namespace", func() {
		DescribeTable("returns a 403 error with invalid operation",
			func(http_method string) {
				client, token, _ := k8s_api.GetK8sClient()
				url := k8s_api.API_SERVER + "/apps/pods"

				req, err := http.NewRequest(http_method, url, nil)
				if err != nil {
					log.Fatal(err)
				}
				req.Header.Set("Authorization", "Bearer "+token)
				req.Header.Set("Accept", "application/yaml")

				resp, err := client.Do(req)
				if err != nil {
					err = fmt.Errorf("Error got %v status, retrieving %v with %v", resp.StatusCode, url, http_method)
				}
				defer resp.Body.Close()
				Expect(resp.StatusCode).To(Equal(403))
			},
			Entry("for DELETE", "DELETE"),
			Entry("for PATCH", "PATCH"),
			Entry("for PUT", "PUT"),
			Entry("for POST", "POST"),
		)
	})
})
