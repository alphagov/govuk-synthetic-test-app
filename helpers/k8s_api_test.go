package helpers_test

import (
	"context"
	"fmt"
	"net/http"

	k8s_api "github.com/alphagov/govuk-synthetic-test-app/helpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Synthetic Test Assumed role", func() {
	Context("when calling k8s api with apps namespace and pods kind", func() {
		It("returns pods list and can access the first image value", func(ctx SpecContext) {
			aws_account_id, err := k8s_api.GetAwsAccountID(ctx)
			Expect(err).NotTo(HaveOccurred())
			podList, _ := k8s_api.GetPodList(ctx, aws_account_id, k8s_api.CLUSTER_ID, k8s_api.ASSUME_ROLE_NAME, "apps")
			GinkgoWriter.Printf("First pod image: %s, %s\n", podList.Items[0].Labels["app"], podList.Items[0].Spec.Containers[0].Image)
			Expect(podList.Items[0].Spec.Containers[0].Image).NotTo(BeNil())
		})
	})

	Context("when trying to perform a DELETE, PATCH, POST, PUT with the k8s api on the apps namespace", func() {
		DescribeTable("returns a 403 error with invalid operation",
			func(ctx SpecContext, http_method string) {
				aws_account_id, err := k8s_api.GetAwsAccountID(ctx)
				Expect(err).NotTo(HaveOccurred())
				client, err := k8s_api.GetK8sClient(ctx, aws_account_id, k8s_api.CLUSTER_ID, k8s_api.ASSUME_ROLE_NAME)
				Expect(err).NotTo(HaveOccurred())

				url := client.ClusterEndpoint + "/api/v1/namespaces/apps/pods"
				req, err := http.NewRequest(http_method, url, nil)
				Expect(err).NotTo(HaveOccurred())

				req.Header.Set("Authorization", "Bearer "+client.Token)
				req.Header.Set("Accept", "application/yaml")

				resp, err := client.Client.Do(req)
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

	Context("when calling k8s api from the production account", func() {
		BeforeEach(func() {
			ctx := context.TODO()
			aws_account_id, err := k8s_api.GetAwsAccountID(ctx)
			Expect(err).NotTo(HaveOccurred())
			if aws_account_id != k8s_api.PRODUCTION_AWS_ACCOUNT_ID {
				Skip("Not in production account")
			}
		})

		DescribeTable("it can assume the synthetic test assumed role in other accounts",
			func(ctx SpecContext, environment_account_id string) {
				podList, _ := k8s_api.GetPodList(ctx, environment_account_id, k8s_api.CLUSTER_ID, k8s_api.ASSUME_ROLE_NAME, "apps")
				Expect(podList.Items[0].Spec.Containers[0].Image).NotTo(BeNil())
			},
			Entry("for integration", k8s_api.INTEGRATION_AWS_ACCOUNT_ID),
			Entry("for staging", k8s_api.STAGING_AWS_ACCOUNT_ID),
			Entry("for production", k8s_api.PRODUCTION_AWS_ACCOUNT_ID),
		)
	})
})
