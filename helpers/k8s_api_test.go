package helpers_test

import (
	"fmt"
	"net/http"
	"os"

	k8s_api "github.com/alphagov/govuk-synthetic-test-app/helpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Synthetic Test Assumed role", func() {
	Context("when calling k8s api with apps namespace and pods kind", func() {
		It("returns pods list and can access the first image value", func(ctx SpecContext) {
			podList, _ := k8s_api.GetPodList(ctx, os.Getenv("ENVIRONMENT_NAME"), "apps")
			GinkgoWriter.Printf("First pod image: %s, %s\n", podList.Items[0].Labels["app"], podList.Items[0].Spec.Containers[0].Image)
			Expect(podList.Items[0].Spec.Containers[0].Image).NotTo(BeNil())
		})
		It("returns pods list and all pods are running with arch arm64", func(ctx SpecContext) {
			podList, _ := k8s_api.GetPodList(ctx, os.Getenv("ENVIRONMENT_NAME"), "apps")
			Expect(podList.Items[0].Labels["app.kubernetes.io/arch"]).To(Equal("arm64"))

			for _, item := range podList.Items {
				Expect(item.Labels).To(
					HaveKeyWithValue("app.kubernetes.io/arch", "arm64"),
					fmt.Sprintf("item %s is missing the app.kubernetes.io/arch label, or its value isn't 'arm64'", item.Name),
				)
			}
		})
	})

	Context("when calling k8s api with NON 'apps' namespace and `pods` kind", func() {
		DescribeTable("assumed role cannot access the namespace",
			func(ctx SpecContext, namespace string) {
				podList, err := k8s_api.GetPodList(ctx, os.Getenv("ENVIRONMENT_NAME"), namespace)
				Expect(podList).To(BeNil())
				Expect(err).To(HaveOccurred())
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
			func(ctx SpecContext, http_method string) {
				client, token, _ := k8s_api.GetK8sClient(ctx, os.Getenv("ENVIRONMENT_NAME"))
				url := k8s_api.API_SERVER + "/apps/pods"

				req, err := http.NewRequest(http_method, url, nil)
				Expect(err).NotTo(HaveOccurred())

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

	if os.Getenv("ENVIRONMENT_NAME") == "production" {
		Context("when calling k8s api from the production account", func() {
			DescribeTable("it can assume the synthetic test assumed role in other accounts",
				func(ctx SpecContext, environment string) {
					podList, _ := k8s_api.GetPodList(ctx, environment, "apps")
					Expect(podList.Items[0].Spec.Containers[0].Image).NotTo(BeNil())
				},
				Entry("for integration", k8s_api.INTEGRATION),
				Entry("for staging", k8s_api.STAGING),
				Entry("for production", k8s_api.PRODUCTION),
			)
		})
	} else if os.Getenv("ENVIRONMENT_NAME") == "staging" {
		Context("when calling k8s api from the staging account", func() {
			DescribeTable("it can't assume the synthetic test assumed role in other accounts",
				func(ctx SpecContext, environment string) {
					podList, err := k8s_api.GetPodList(ctx, environment, "apps")
					Expect(podList).To(BeNil())
					Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("User: arn:aws:sts::%s:assumed-role/synthetic-test-assumer", k8s_api.STAGING_AWS_ACCOUNT_ID)))
					environment_account_id, env_err := k8s_api.GetEnvironmentAccountID(environment)
					Expect(env_err).To(BeNil())
					Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("is not authorized to perform: sts:AssumeRole on resource: arn:aws:iam::%s:role/synthetic-test-assumed", environment_account_id)))
				},
				Entry("for integration", k8s_api.INTEGRATION),
				Entry("for production", k8s_api.PRODUCTION),
			)
		})
	} else if os.Getenv("ENVIRONMENT_NAME") == "integration" {
		Context("when calling k8s api from the integration account", func() {
			DescribeTable("it can't assume the synthetic test assumed role in other accounts",
				func(ctx SpecContext, environment string) {
					podList, err := k8s_api.GetPodList(ctx, environment, "apps")
					Expect(podList).To(BeNil())
					Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("User: arn:aws:sts::%s:assumed-role/synthetic-test-assumer", k8s_api.INTEGRATION_AWS_ACCOUNT_ID)))
					environment_account_id, env_err := k8s_api.GetEnvironmentAccountID(environment)
					Expect(env_err).To(BeNil())
					Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("is not authorized to perform: sts:AssumeRole on resource: arn:aws:iam::%s:role/synthetic-test-assumed", environment_account_id)))
				},
				Entry("for staging", k8s_api.STAGING),
				Entry("for production", k8s_api.PRODUCTION),
			)
		})
	}
})
