package helpers_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/alphagov/govuk-synthetic-test-app/helpers"
)

var _ = Describe("ArgoCD and GitHub Image Sync. Given the canary application it should be synced, healthy, and running the latest version from GitHub tag / container sha", Ordered, func() {
	const (
		namespace        = "apps"
		appName          = "govuk-synthetic-test-app-canary"
		applicationCrdNs = "cluster-services"
		repo             = "alphagov/govuk-synthetic-test-app-canary"
		appLabelSelector = "app=govuk-synthetic-test-app-canary"
		containerPath    = "alphagov/govuk/govuk-synthetic-test-app-canary"
	)

	var (
		digest    string
		token     string
		latestTag string
		err       error
	)

	BeforeAll(func(ctx context.Context) {
		token, err = helpers.GetGHCRToken(ctx, containerPath)
		Expect(err).NotTo(HaveOccurred())

		latestTag, err = helpers.GetLatestGitHubReleaseTag(ctx, repo)
		Expect(err).NotTo(HaveOccurred())
		fmt.Printf("Latest GitHub release tag: %s\n", latestTag)

		digest, err = helpers.GetGHCRImageDigest(ctx, containerPath, latestTag, token)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should be able to query argo and the cluster from the same cluster the tests are ran from (production)", func(ctx context.Context) {
		prodClient, err := helpers.GetK8sClient(ctx, helpers.PRODUCTION_AWS_ACCOUNT_ID, helpers.CLUSTER_ID, helpers.ASSUME_ROLE_NAME)
		Expect(err).NotTo(HaveOccurred())

		Expect(prodClient).NotTo(BeNil())

		isSyncedAndHealthy, err := helpers.GetArgoCDApplicationStatus(ctx, prodClient, applicationCrdNs, appName)
		Expect(err).NotTo(HaveOccurred())
		Expect(isSyncedAndHealthy).To(BeTrue(), "ArgoCD application %s should be Synced and Healthy", appName)

		tag, sha, err := helpers.GetPodImageDetails(ctx, prodClient, namespace, appLabelSelector)
		Expect(err).NotTo(HaveOccurred())
		fmt.Printf("Deployed image tag: %s, SHA: %s\n", tag, sha)

		Expect(tag).To(Equal(latestTag), "Deployed image tag %s does not match latest GitHub release tag %s", tag, latestTag)

		Expect(sha).To(Equal(digest), "Deplod sha does not match the latest digest pulled from ghcr", sha, digest)
	})

	It("should be able to query a different cluster (staging) by assuming a different role", func(ctx context.Context) {
		stagingClient, err := helpers.GetK8sClient(ctx, helpers.STAGING_AWS_ACCOUNT_ID, helpers.CLUSTER_ID, helpers.ASSUME_ROLE_NAME)
		Expect(err).NotTo(HaveOccurred())

		Expect(stagingClient).NotTo(BeNil())

		isSyncedAndHealthy, err := helpers.GetArgoCDApplicationStatus(ctx, stagingClient, applicationCrdNs, appName)
		Expect(err).NotTo(HaveOccurred())
		Expect(isSyncedAndHealthy).To(BeTrue(), "ArgoCD application %s should be Synced and Healthy", appName)

		tag, sha, err := helpers.GetPodImageDetails(ctx, stagingClient, namespace, appLabelSelector)
		Expect(err).NotTo(HaveOccurred())
		fmt.Printf("Deployed image tag: %s, SHA: %s\n", tag, sha)

		Expect(tag).To(Equal(latestTag), "Deployed image tag %s does not match latest GitHub release tag %s", tag, latestTag)

		Expect(sha).To(Equal(digest), "Deplod sha does not match the latest digest pulled from ghcr", sha, digest)
	})

	FIt("should be able to query a different cluster (integration) by assuming a different role", func(ctx context.Context) {
		stagingClient, err := helpers.GetK8sClient(ctx, helpers.INTEGRATION_AWS_ACCOUNT_ID, helpers.CLUSTER_ID, helpers.ASSUME_ROLE_NAME)
		Expect(err).NotTo(HaveOccurred())

		Expect(stagingClient).NotTo(BeNil())

		isSyncedAndHealthy, err := helpers.GetArgoCDApplicationStatus(ctx, stagingClient, applicationCrdNs, appName)
		Expect(err).NotTo(HaveOccurred())
		Expect(isSyncedAndHealthy).To(BeTrue(), "ArgoCD application %s should be Synced and Healthy", appName)

		tag, sha, err := helpers.GetPodImageDetails(ctx, stagingClient, namespace, appLabelSelector)
		Expect(err).NotTo(HaveOccurred())
		fmt.Printf("Deployed image tag: %s, SHA: %s\n", tag, sha)

		Expect(tag).To(Equal(latestTag), "Deployed image tag %s does not match latest GitHub release tag %s", tag, latestTag)

		Expect(sha).To(Equal(digest), "Deplod sha does not match the latest digest pulled from ghcr", sha, digest)
	})
})
