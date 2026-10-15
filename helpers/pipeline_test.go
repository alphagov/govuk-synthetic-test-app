package helpers_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"

	"github.com/alphagov/govuk-synthetic-test-app/helpers"
)

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
	digest                        string
	latestTag                     string
	deploymentPipelineTestSuccess *prometheus.GaugeVec
	promReg                       *prometheus.Registry
)

func buildDeployLabels(tag, env string) string {
	return fmt.Sprintf("repoName=govuk-synthetic-test-app-canary,imageTag=%s,workflows.argoproj.io/workflow-template=deploy-image,environment=%s", tag, env)
}

func buildPostSyncLabels(tag string) string {
	return fmt.Sprintf("repoName=govuk-synthetic-test-app-canary,imageTag=%s,workflows.argoproj.io/workflow-template=post-sync", tag)
}

var _ = BeforeSuite(func(ctx SpecContext) {
	By("Configuring git")
	err := helpers.ConfigureGit(ctx)
	Expect(err).NotTo(HaveOccurred())

	promReg = prometheus.NewRegistry()
	deploymentPipelineTestSuccess = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "deployment_pipeline_test_success",
		Help: "Status of deployment pipeline test execution",
	},
		[]string{"environment"},
	)
	promReg.MustRegister(deploymentPipelineTestSuccess)
})

var _ = Describe("GIVEN the Argo + Github deployment pipeline THEN the canary application SHOULD be synced, healthy, and running the latest version from GitHub tag / container sha", Ordered, func() {
	verifyReleaseWorkflow := func(g Gomega, ctx context.Context, headSha, token string) {
		releaseResp, err := helpers.GetGithubWorkflowRunStatus(ctx, "release.yml", headSha, token)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(releaseResp.WorkflowRuns[0].Status).To(Equal("completed"), "Release workflow completed")
		g.Expect(releaseResp.WorkflowRuns[0].Conclusion).To(Equal("success"), "Release workflow successful")
	}

	verifyDeployWorkflow := func(g Gomega, ctx context.Context, headSha, token string) {
		deployResp, err := helpers.GetGithubWorkflowRunStatus(ctx, "deploy.yml", headSha, token)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(deployResp.WorkflowRuns[0].Status).To(Equal("completed"), "Deploy workflow completed")
		g.Expect(deployResp.WorkflowRuns[0].Conclusion).To(Equal("success"), "Deploy workflow successful")
	}

	verifyPostSyncWorkflow := func(g Gomega, ctx context.Context, envClient *helpers.K8sClient, latestTag, appsNs string) {
		postSyncLabel := buildPostSyncLabels(latestTag)

		postSyncOk, err := helpers.GetArgoWorkflowStatus(ctx, envClient, appsNs, postSyncLabel)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(postSyncOk).To(BeTrue(), "Argo Post Sync workflow Failed")
	}

	verifyDeployImageWorkflow := func(g Gomega, ctx context.Context, prodClient *helpers.K8sClient, latestTag, appNs, env string) {
		deployLabel := buildDeployLabels(latestTag, env)

		deployOk, err := helpers.GetArgoWorkflowStatus(ctx, prodClient, appsNs, deployLabel)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(deployOk).To(BeTrue(), "Argo Deploy Image workflow Failed")

		sourceImageTagVal, err := helpers.GetImageTagFromChartRepo(ctx, env)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(sourceImageTagVal).To(Equal(latestTag), "The value in github source code for the %s environment _does not match_ the latest release tag %s", env, latestTag)
	}

	verifyPodImage := func(g Gomega, ctx context.Context, envClient *helpers.K8sClient, appsNs, appLabelSelector, env string) {
		tag, sha, err := helpers.GetPodImageDetails(ctx, envClient, appsNs, appLabelSelector)
		g.Expect(err).NotTo(HaveOccurred())

		By("[" + env + "] Deployed image tag: " + tag + ", SHA: +" + sha)

		g.Expect(tag).To(Equal(latestTag), "Deployed image tag %s _does not match_ latest GitHub release tag %s", tag, latestTag)

		g.Expect(sha).To(Equal(digest), "Deplod sha _does not match_ the latest digest pulled from ghcr", sha, digest)
	}

	verifyVersionDisplayedInApp := func(g Gomega, ctx context.Context, latestTag string) {
		displayedVersion, err := helpers.GetVersionFromApp(ctx)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(displayedVersion).To(Equal(latestTag))
	}

	BeforeAll(func(ctx SpecContext) {
		By("[BeforeAll] Triggering the deployment by committing and releasing a new app version")
		githubAccessToken, err := helpers.GetGitHubAppToken(ctx, os.Getenv("GITHUB_APP_ID"), os.Getenv("GITHUB_INSTALL_ID"), os.Getenv("GITHUB_PEM_STR"))

		headSha, err := helpers.IncrementCanaryVersion(ctx, "https://github.com/alphagov/govuk-synthetic-test-app-canary", githubAccessToken)
		Expect(err).NotTo(HaveOccurred())

		trimmedHeadSha := strings.TrimSpace(headSha)

		token, err := helpers.GetGHCRToken(ctx, containerPath)
		Expect(err).NotTo(HaveOccurred())

		By("[BeforeAll] Querying status of Canary App in the Release GitHub Action")
		Eventually(verifyReleaseWorkflow).
			WithContext(ctx).
			WithArguments(trimmedHeadSha, githubAccessToken).
			WithTimeout(5 * time.Minute).
			WithPolling(30 * time.Second).
			Should(Succeed())

		By("[BeforeAll] Querying status of Canary App in the Deploy GitHub Action")
		Eventually(verifyDeployWorkflow).
			WithContext(ctx).
			WithArguments(trimmedHeadSha, githubAccessToken).
			WithTimeout(5 * time.Minute).
			WithPolling(30 * time.Second).
			Should(Succeed())

		latestTag, err = helpers.GetLatestGitHubReleaseTag(ctx, repo)
		Expect(err).NotTo(HaveOccurred())
		By("[BeforeAll] Latest GitHub release tag: " + latestTag)

		digest, err = helpers.GetGHCRImageDigest(ctx, containerPath, latestTag, token)
		Expect(err).NotTo(HaveOccurred())
		By("[BeforeAll] Latest GitHub container digest: " + digest)
	})

	DescribeTable("Query each state of the deployment pipeline", Ordered,
		func(ctx SpecContext, env string) {
			AddReportEntry("TargetEnvironment", env)
			By("[" + env + "] Bootstrapping k8s clients for different environments")
			var envClient *helpers.K8sClient

			prodClient, err := helpers.GetK8sClient(ctx, helpers.PRODUCTION_AWS_ACCOUNT_ID, helpers.CLUSTER_ID, helpers.ASSUME_ROLE_NAME)
			Expect(err).NotTo(HaveOccurred())
			Expect(prodClient).NotTo(BeNil())

			switch env {
			case integration:
				envClient, err = helpers.GetK8sClient(ctx, helpers.INTEGRATION_AWS_ACCOUNT_ID, helpers.CLUSTER_ID, helpers.ASSUME_ROLE_NAME)
				Expect(err).NotTo(HaveOccurred())
				Expect(envClient).NotTo(BeNil())
			case staging:
				envClient, err = helpers.GetK8sClient(ctx, helpers.STAGING_AWS_ACCOUNT_ID, helpers.CLUSTER_ID, helpers.ASSUME_ROLE_NAME)
				Expect(err).NotTo(HaveOccurred())
				Expect(envClient).NotTo(BeNil())
			case production:
				envClient = prodClient
			}

			By("[" + env + "] Checking the status of the deployment")

			Eventually(verifyPostSyncWorkflow).
				WithContext(ctx).
				WithArguments(envClient, latestTag, appsNs).
				WithTimeout(15 * time.Minute).
				WithPolling(1 * time.Minute).
				Should(Succeed())

			By("[" + env + "] Argo Workflow Post Sync is successful")

			Eventually(verifyDeployImageWorkflow).
				WithContext(ctx).
				WithArguments(prodClient, latestTag, appsNs, env).
				WithTimeout(15 * time.Minute).
				WithPolling(1 * time.Minute).
				Should(Succeed())

			By("[" + env + "] Argo Workflow Deploy Image is successful")

			isSyncedAndHealthy, err := helpers.GetArgoCDApplicationStatus(ctx, envClient, applicationCrdNs, appName)
			Expect(err).NotTo(HaveOccurred())
			Expect(isSyncedAndHealthy).To(BeTrue(), "ArgoCD application %s is not Synced and Healthy", appName)

			By("[" + env + "] Argo CD govuk-synthetic-test-app-canary is synced and healthy")

			Eventually(verifyPodImage).
				WithContext(ctx).
				WithArguments(envClient, appsNs, appLabelSelector, env).
				WithTimeout(10 * time.Minute).
				WithPolling(1 * time.Minute).
				Should(Succeed())

			By("[" + env + "] Canary App release has rolled out successfully and the deployed tag and sha are correct")

			Eventually(verifyVersionDisplayedInApp).
				WithContext(ctx).
				WithArguments(latestTag).
				WithTimeout(10 * time.Minute).
				WithPolling(1 * time.Minute).
				Should(Succeed())

			By("[" + env + "] Application is serving the new data")
		},

		Entry(
			"WHEN the environment is INTEGRATION",
			integration,
		),

		Entry(
			"WHEN the environment is STAGING",
			staging,
		),

		Entry(
			"WHEN the environment is PRODUCTION",
			production,
		),
	)

	ReportAfterEach(func(report SpecReport) {
		for _, entry := range report.ReportEntries {
			if entry.Name == "TargetEnvironment" {
				env := entry.Value.GetRawValue().(string)
				By("Setting up Prometheus Pushgateway for " + env)

				prometheusPushGatewayUrl := os.Getenv("PROMETHEUS_PUSHGATEWAY_URL")

				if report.Failed() {
					deploymentPipelineTestSuccess.WithLabelValues(env).Set(0)
				} else {
					deploymentPipelineTestSuccess.WithLabelValues(env).Set(1)
				}

				By("Pushing deployment_pipeline_test_success_metrics to Prometheus Pushgateway for " + env)
				err := push.New(prometheusPushGatewayUrl, "deployment_pipeline_test_metrics").Gatherer(promReg).Push()
				Expect(err).NotTo(HaveOccurred(), "Error pushing metrics to Prometheus Pushgateway")

			}
		}
	})
})
