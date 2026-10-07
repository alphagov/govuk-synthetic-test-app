package helpers_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/alphagov/govuk-synthetic-test-app/helpers"
)

func buildDeployLabels(tag, env string) string {
	return fmt.Sprintf("repoName=govuk-synthetic-test-app-canary,imageTag=%s,workflows.argoproj.io/workflow-template=deploy-image,environment=%s", tag, env)
}

func buildPostSyncLabels(tag string) string {
	return fmt.Sprintf("repoName=govuk-synthetic-test-app-canary,imageTag=%s,workflows.argoproj.io/workflow-template=post-sync", tag)
}

var _ = Describe("ArgoCD and GitHub Image Sync. Given the canary application it should be synced, healthy, and running the latest version from GitHub tag / container sha", Ordered, func() {
	SetDefaultEventuallyTimeout(10 * time.Minute)
	SetDefaultEventuallyPollingInterval(10 * time.Second)

	const (
		appsNs           = "apps"
		appName          = "govuk-synthetic-test-app-canary"
		applicationCrdNs = "cluster-services"
		repo             = "alphagov/govuk-synthetic-test-app-canary"
		appLabelSelector = "app=govuk-synthetic-test-app-canary"
		containerPath    = "alphagov/govuk/govuk-synthetic-test-app-canary"
		integration      = "integration"
		staging          = "staging"
		production       = "production"
	)

	var (
		digest        string
		token         string
		latestTag     string
		intClient     *helpers.K8sClient
		stagingClient *helpers.K8sClient
		prodClient    *helpers.K8sClient
		err           error
	)

	BeforeAll(func(ctx SpecContext) {
		// TODO: create new release / or bump .version file by pushing to main
		token, err = helpers.GetGHCRToken(ctx, containerPath)
		Expect(err).NotTo(HaveOccurred())

		latestTag, err = helpers.GetLatestGitHubReleaseTag(ctx, repo)
		Expect(err).NotTo(HaveOccurred())
		fmt.Printf("Latest GitHub release tag: %s\n", latestTag)

		digest, err = helpers.GetGHCRImageDigest(ctx, containerPath, latestTag, token)
		Expect(err).NotTo(HaveOccurred())

		intClient, err = helpers.GetK8sClient(ctx, helpers.INTEGRATION_AWS_ACCOUNT_ID, helpers.CLUSTER_ID, helpers.ASSUME_ROLE_NAME)
		Expect(err).NotTo(HaveOccurred())
		Expect(intClient).NotTo(BeNil())

		stagingClient, err = helpers.GetK8sClient(ctx, helpers.STAGING_AWS_ACCOUNT_ID, helpers.CLUSTER_ID, helpers.ASSUME_ROLE_NAME)
		Expect(err).NotTo(HaveOccurred())
		Expect(intClient).NotTo(BeNil())

		prodClient, err = helpers.GetK8sClient(ctx, helpers.PRODUCTION_AWS_ACCOUNT_ID, helpers.CLUSTER_ID, helpers.ASSUME_ROLE_NAME)
		Expect(err).NotTo(HaveOccurred())
		Expect(intClient).NotTo(BeNil())
	})

	DescribeTable("Extracting the author's first and last name", Ordered,
		func(ctx SpecContext, envClient *helpers.K8sClient, env string) {
			verifyPostSyncWorkflow := func(g Gomega) {
				postSyncLabel := buildPostSyncLabels(latestTag)

				postSyncOk, err := helpers.GetArgoWorkflowStatus(ctx, envClient, appsNs, postSyncLabel)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(postSyncOk).To(BeTrue(), "Argo Post Sync workflow Succeeded")
			}

			verifyDeployImageWorkflow := func(g Gomega) {
				deployLabel := buildDeployLabels(latestTag, env)

				deployOk, err := helpers.GetArgoWorkflowStatus(ctx, prodClient, appsNs, deployLabel)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(deployOk).To(BeTrue(), "Argo Deploy Image workflow Succeeded")

				sourceImageTagVal, err := helpers.GetImageTagFromChartRepo(env)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(sourceImageTagVal).To(Equal(latestTag), "The value in github source for the %s environment matches the latest release tag %s", env, latestTag)
			}

			Eventually(ctx, verifyPostSyncWorkflow).Should(Succeed())
			Eventually(ctx, verifyDeployImageWorkflow).Should(Succeed())

			isSyncedAndHealthy, err := helpers.GetArgoCDApplicationStatus(ctx, envClient, applicationCrdNs, appName)
			Expect(err).NotTo(HaveOccurred())
			Expect(isSyncedAndHealthy).To(BeTrue(), "ArgoCD application %s should be Synced and Healthy", appName)

			tag, sha, err := helpers.GetPodImageDetails(ctx, envClient, appsNs, appLabelSelector)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("Deployed image tag: %s, SHA: %s\n", tag, sha)

			Expect(tag).To(Equal(latestTag), "Deployed image tag %s does not match latest GitHub release tag %s", tag, latestTag)

			Expect(sha).To(Equal(digest), "Deplod sha does not match the latest digest pulled from ghcr", sha, digest)
			// TODO: hit the actual app endpoint directly and verify it displays the correct version
		},

		Entry("WHEN the environment is INTEGRATION", intClient, integration),
		Entry("WHEN the environment is STAGING", stagingClient, staging),
		Entry("WHEN the environment is PRODUCTION", prodClient, production),
	)
})
