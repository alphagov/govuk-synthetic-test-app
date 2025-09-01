package govuk_synthetic_test_app_test

import (
	"io"
	"net/http"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Deploy", func() {
	host := "localhost:3000"
	It("should respond OK to a GET request", func() {
		resp, err := http.Get("http://" + host + "?status=200")
		if err != nil {
			Fail(err.Error())
		}
		defer resp.Body.Close()

		Expect(resp.StatusCode).To(Equal(200))
	})

	It("should respond with the version matching the file", func() {
		resp, err := http.Get("http://" + host + "/version")
		if err != nil {
			Fail(err.Error())
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				Fail(err.Error())
			}
			bodyString := string(bodyBytes)

			file, err := os.Open(".version")
			if err != nil {
				Fail(err.Error())
			}

			version, err := io.ReadAll(file)
			if err != nil {
				Fail(err.Error())
			}

			Expect(bodyString).To(Equal(string(version)))
		}
	})
})
