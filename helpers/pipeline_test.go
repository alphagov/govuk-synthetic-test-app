package helpers_test

import (
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/alphagov/govuk-synthetic-test-app/helpers"
)

var (
	intClient     *helpers.K8sClient
	stagingClient *helpers.K8sClient
	prodClient    *helpers.K8sClient
)

func buildDeployLabels(tag, env string) string {
	return fmt.Sprintf("repoName=govuk-synthetic-test-app-canary,imageTag=%s,workflows.argoproj.io/workflow-template=deploy-image,environment=%s", tag, env)
}

func buildPostSyncLabels(tag string) string {
	return fmt.Sprintf("repoName=govuk-synthetic-test-app-canary,imageTag=%s,workflows.argoproj.io/workflow-template=post-sync", tag)
}

var _ = BeforeSuite(func(ctx SpecContext) {
	By("bootstrapping k8s clients for different environments")
	var err error

	intClient, err = helpers.GetK8sClient(ctx, helpers.INTEGRATION_AWS_ACCOUNT_ID, helpers.CLUSTER_ID, helpers.ASSUME_ROLE_NAME)
	Expect(err).NotTo(HaveOccurred())
	Expect(intClient).NotTo(BeNil())

	stagingClient, err = helpers.GetK8sClient(ctx, helpers.STAGING_AWS_ACCOUNT_ID, helpers.CLUSTER_ID, helpers.ASSUME_ROLE_NAME)
	Expect(err).NotTo(HaveOccurred())
	Expect(stagingClient).NotTo(BeNil())

	prodClient, err = helpers.GetK8sClient(ctx, helpers.PRODUCTION_AWS_ACCOUNT_ID, helpers.CLUSTER_ID, helpers.ASSUME_ROLE_NAME)
	Expect(err).NotTo(HaveOccurred())
	Expect(prodClient).NotTo(BeNil())
})

var _ = FDescribe("GIVEN the Argo + Github deployment pipeline THEN the canary application SHOULD be synced, healthy, and running the latest version from GitHub tag / container sha", Ordered, func() {
	SetDefaultEventuallyTimeout(2 * time.Minute)
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
		digest            string
		token             string
		latestTag         string
		headSha           string
		githubAccessToken string
		err               error
	)

	BeforeAll(func(ctx SpecContext) {
		// TODO: weave context with cancel into all the functions
		githubAccessToken = os.Getenv("GITHUB_ACCESS_TOKEN")

		headSha, err = helpers.IncrementCanaryVersion(ctx, "https://github.com/alphagov/govuk-synthetic-test-app-canary", githubAccessToken)
		Expect(err).NotTo(HaveOccurred())

		token, err = helpers.GetGHCRToken(ctx, containerPath)
		Expect(err).NotTo(HaveOccurred())

		latestTag, err = helpers.GetLatestGitHubReleaseTag(ctx, repo)
		Expect(err).NotTo(HaveOccurred())
		fmt.Printf("Latest GitHub release tag: %s\n", latestTag)

		digest, err = helpers.GetGHCRImageDigest(ctx, containerPath, latestTag, token)
		Expect(err).NotTo(HaveOccurred())
		fmt.Printf("Latest GitHub container digest: %s\n", digest)
	})

	DescribeTable("Query each state of the deployment pipeline", Ordered,
		func(ctx SpecContext, getEnvClient, getProdClient func() *helpers.K8sClient, env string) {
			envClient := getEnvClient()
			prodClient := getProdClient()

			verifyGithubWorkflow := func(g Gomega) {
				releaseResp, err := helpers.GetGithubWorkflowRunStatus(ctx, "release.yml", headSha, githubAccessToken)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(releaseResp.WorkflowRuns[0].Status).To(Equal("completed"), "Release workflow completed")
				g.Expect(releaseResp.WorkflowRuns[0].Conclusion).To(Equal("success"), "Release workflow successful")

				deployResp, err := helpers.GetGithubWorkflowRunStatus(ctx, "deploy.yml", headSha, githubAccessToken)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(deployResp.WorkflowRuns[0].Status).To(Equal("completed"), "Deploy workflow completed")
				g.Expect(deployResp.WorkflowRuns[0].Conclusion).To(Equal("success"), "Deploy workflow successful")
			}

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

			Eventually(ctx, verifyGithubWorkflow).Should(Succeed())
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

			displayedVersion, err := helpers.GetVersionFromApp(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(displayedVersion).To(Equal(latestTag))
		},

		Entry(
			"WHEN the environment is INTEGRATION",
			func() *helpers.K8sClient { return intClient },
			func() *helpers.K8sClient { return prodClient },
			integration,
		),

		Entry(
			"WHEN the environment is STAGING",
			func() *helpers.K8sClient { return stagingClient },
			func() *helpers.K8sClient { return prodClient },
			staging,
		),

		Entry(
			"WHEN the environment is PRODUCTION",
			func() *helpers.K8sClient { return prodClient },
			func() *helpers.K8sClient { return prodClient },
			production,
		),
	)
})
