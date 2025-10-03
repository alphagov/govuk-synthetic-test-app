package helpers_test

import (
	"fmt"
	k8s_api "govuk-synthetic-test-app/helpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AssumeRole", func() {
	Describe("AssumeRole", func() {
		Context("when called with apps namespace and pods kind", func() {
			It("returns pods list", func() {
				fmt.Println("==== apps pod list ====")
				podList, _ := k8s_api.GetPodList("apps", "pods")
				fmt.Printf("Pods: %+v, %+v\n", podList.Items[0].Labels["app"], podList.Items[0].Spec.Containers[0].Image)

				Expect(podList).To(Equal("AssumeRole"))
			})
		})
	})
})
