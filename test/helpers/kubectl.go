package helpers

import (
	"encoding/json"
	"log"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/gruntwork-io/terratest/modules/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type AppJson struct {
	Status struct {
		Sync struct {
			Status string `json:"status"`
		} `json:"sync"`
		Health struct {
			Status string `json:"status"`
		} `json:"health"`
	} `json:"status"`
}

func WaitingForHealthStatus(g Gomega, gingko testing.TestingT, kubeContext, kubeConfigPath, healthStatus string) string {
	options := k8s.NewKubectlOptions(kubeContext, kubeConfigPath, "cluster-services")
	options.Logger = logger.Discard
	app, err := k8s.RunKubectlAndGetOutputE(GinkgoT(), options, "get", "applications.argoproj.io", "govuk-synthetic-test-app", "-ojson")

	options.Logger = logger.Discard
	var appJson AppJson
	unmarshallErr := json.Unmarshal([]byte(app), &appJson)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(unmarshallErr).ToNot(HaveOccurred())
	g.Expect(appJson.Status.Sync.Status).To(Equal("Synced"))
	log.Print("Waiting for '"+healthStatus+"' health status, current status is: ", appJson.Status.Health.Status)
	return appJson.Status.Health.Status
}
