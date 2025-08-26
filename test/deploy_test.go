package govuk_synthetic_test_app_test

import (
	"encoding/json"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/alphagov/govuk-synthetic-test-app/test/helpers"
	"github.com/google/go-github/v74/github"
	"github.com/gruntwork-io/terratest/modules/k8s"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type PodJson struct {
	Items []struct {
		Spec struct {
			Containers []struct {
				Image string `json:"image"`
			} `json:"containers"`
		} `json:"spec"`
		Status struct {
			ContainerStatuses []struct {
				Image   string `json:"image"`
				ImageId string `json:"imageID"`
				State   struct {
					Running struct {
						StartedAt string `json:"startedAt,omitempty"`
					} `json:"running"`
				} `json:"State"`
			} `json:"containerStatuses"`
		} `json:"status"`
	} `json:"items"`
}

var _ = Describe("GIVEN an application deployment pipeline commit to main THEN deploy to integration", Ordered, func() {
	// TODO: move this into a config file, so we can pass these vals in as flags
	ghAuth, ghAuthPresent := os.LookupEnv("GITHUB_TOKEN")
	if ghAuth == "" || !ghAuthPresent {
		log.Fatal("GITHUB_TOKEN is not set")
	}

	kubeContext, kubeContextPresent := os.LookupEnv("KUBE_CONTEXT")
	if kubeContext == "" || !kubeContextPresent {
		log.Fatal("KUBE_CONTEXT is not set")
	}

	kubeConfigPath, kubeConfigPathPresent := os.LookupEnv("KUBECONFIG_PATH")
	if kubeConfigPath == "" || !kubeConfigPathPresent {
		log.Fatal("KUBECONFIG_PATH is not set")
	}

	date := time.Now()

	author := &github.CommitAuthor{Date: &github.Timestamp{Time: date}, Name: github.Ptr("jaskaran"), Email: github.Ptr("jaskaran.sarkaria@example.gov.uk")}

	client := github.NewClient(nil).WithAuthToken(ghAuth)

	Context("WHEN a commit to main is made", func() {
		It("THEN the expected version should be released to integration AND THEN to production", func(ctx SpecContext) {
			latestTag, err := helpers.GetCurrentRelease(ctx, client)
			Expect(err).ToNot(HaveOccurred())
			trimmed := strings.TrimPrefix(*latestTag.TagName, "v")
			i, err := strconv.Atoi(trimmed)
			Expect(err).ToNot(HaveOccurred())

			i += 1

			expectedVersion := "v" + strconv.Itoa(i)

			commitErr := helpers.CommitVersionChange(ctx, client, date, expectedVersion, author)
			Expect(commitErr).ToNot(HaveOccurred())

			time.Sleep(3 * time.Minute)
			options := k8s.NewKubectlOptions(kubeContext, kubeConfigPath, "cluster-services")
			_, triggerArgoSyncErr := k8s.RunKubectlAndGetOutputE(GinkgoT(), options, "patch", "applications.argoproj.io", "govuk-synthetic-test-app", "-p", "{\"operation\":{\"initiatedBy\":{\"username\":\"synthetic-deployment-test\"},\"sync\":{\"syncStrategy\":{\"hook\":{}}}}}", "--type", "merge")
			Expect(triggerArgoSyncErr).ToNot(HaveOccurred())

			Eventually(helpers.WaitingForHealthStatus).WithContext(ctx).WithArguments(GinkgoT(), kubeContext, kubeConfigPath, "Progressing").WithPolling(500 * time.Millisecond).Should(Equal("Progressing"))

			Eventually(helpers.WaitingForHealthStatus).WithContext(ctx).WithArguments(GinkgoT(), kubeContext, kubeConfigPath, "Healthy").WithPolling(500 * time.Millisecond).Should(Equal("Healthy"))

			time.Sleep(30 * time.Second)

			appsOptions := k8s.NewKubectlOptions(kubeContext, kubeConfigPath, "apps")
			imageJson, err := k8s.RunKubectlAndGetOutputE(GinkgoT(), appsOptions, "get", "pod", "-l", "app=govuk-synthetic-test-app", "-ojson")

			Expect(err).ToNot(HaveOccurred())
			var podJson PodJson
			unmarshallErr := json.Unmarshal([]byte(imageJson), &podJson)
			Expect(unmarshallErr).ToNot(HaveOccurred())

			Expect(podJson.Items[0].Spec.Containers[0].Image).To(ContainSubstring("dkr.ecr.eu-west-1.amazonaws.com/github/alphagov/govuk/govuk-synthetic-test-app:" + expectedVersion))
			// TODO: look up sha pushed to ecr and match against the ImageId", podJson.Items[0].Status.ContainerStatuses[0].ImageId
			// TODO: check post sync has ran successfully (see google doc for commands)
			// TODO: check production environment has the correct image version (using the same method as above)
			// TODO: on failure push alert to #govuk-deploy-alerts
		}, SpecTimeout(10*time.Minute))
	})
})
