package govuk_synthetic_test_app_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGovukSyntheticTestApp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "GovukSyntheticTestApp Suite")
}
