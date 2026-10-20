package helpers_test

import (
	"net/http"

	k8s_api "github.com/alphagov/govuk-synthetic-test-app/helpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Synthetic Test Assumed role", Ordered, func() {
	Context("when trying to perform a DELETE, PATCH, POST, PUT with the k8s api on the apps namespace", func() {
		DescribeTable("returns a 403 error with invalid operation",
			func(ctx SpecContext, http_method string) {
				awsAccountId, err := k8s_api.GetAwsAccountID(ctx)
				Expect(err).NotTo(HaveOccurred())
				client, err := k8s_api.GetK8sClient(ctx, awsAccountId, k8s_api.CLUSTER_ID, k8s_api.ASSUME_ROLE_NAME)
				Expect(err).NotTo(HaveOccurred())

				url := client.ClusterEndpoint + "/api/v1/namespaces/apps/pods"
				req, err := http.NewRequest(http_method, url, nil)
				Expect(err).NotTo(HaveOccurred())

				req.Header.Set("Authorization", "Bearer "+client.Token)
				req.Header.Set("Accept", "application/yaml")

				resp, err := client.Client.Do(req)
				Expect(err).NotTo(HaveOccurred())

				defer func() {
					closeErr := resp.Body.Close()
					Expect(closeErr).NotTo(HaveOccurred())
				}()

				Expect(resp.StatusCode).To(Equal(403))
			},
			Entry("for DELETE", "DELETE"),
			Entry("for PATCH", "PATCH"),
			Entry("for PUT", "PUT"),
			Entry("for POST", "POST"),
		)
	})

	Context("when calling k8s api from the production account", func() {
		BeforeEach(func(ctx SpecContext) {
			awsAccountId, err := k8s_api.GetAwsAccountID(ctx)
			Expect(err).NotTo(HaveOccurred())
			if awsAccountId != k8s_api.PRODUCTION_AWS_ACCOUNT_ID {
				Skip("Not in production account")
			}
		})

		DescribeTable("it can assume the synthetic test assumed role in other accounts",
			func(ctx SpecContext, envAccountId string) {
				podList, err := k8s_api.GetPodList(ctx, envAccountId, k8s_api.CLUSTER_ID, k8s_api.ASSUME_ROLE_NAME, "apps")
				Expect(err).NotTo(HaveOccurred())
				Expect(podList.Items[0].Spec.Containers[0].Image).NotTo(BeNil())
			},
			Entry("for integration", k8s_api.INTEGRATION_AWS_ACCOUNT_ID),
			Entry("for staging", k8s_api.STAGING_AWS_ACCOUNT_ID),
			Entry("for production", k8s_api.PRODUCTION_AWS_ACCOUNT_ID),
		)
	})
})
