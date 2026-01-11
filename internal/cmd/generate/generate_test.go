package generate_test

import (
	"os"
	"runtime"
	"testing"

	"github.com/jmreicha/lazycfg/internal/cmd/generate"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

func TestGenerate(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Generate Suite")
}

var _ = ginkgo.Describe("Generate", func() {
	ginkgo.AfterEach(func() {
		// Clean up generated files after each test
		_ = os.Remove(generate.GrantedConfigPath)
		_ = os.Remove(generate.GrantedConfigPath + ".test")
	})

	ginkgo.Describe("CreateGrantedConfiguration", func() {
		ginkgo.It("should create a granted config file with OS-specific content", func() {
			// When
			err := generate.CreateGrantedConfiguration(generate.GrantedConfigPath)

			// Then
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(generate.GrantedConfigPath).To(gomega.BeARegularFile())

			// Verify the content of the file
			content, err := os.ReadFile(generate.GrantedConfigPath)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Common configuration should be present regardless of OS
			gomega.Expect(string(content)).To(gomega.ContainSubstring(`DefaultBrowser = "STDOUT"`))
			gomega.Expect(string(content)).To(gomega.ContainSubstring(`DisableUsageTips = true`))

			// OS-specific configuration based on runtime
			switch runtime.GOOS {
			case "darwin":
				// Unimplemented: Check for macOS-specific content
				// Expect(string(content)).To(ContainSubstring("ConsoleUrlBuilderForAwsConsole"))
			case "linux":
				// Unimplemented: Check for Linux-specific content
				// Expect(string(content)).To(ContainSubstring("ConsoleUrlBuilderForAwsConsole"))
			}
		})

		ginkgo.It("should create a config file at a custom location", func() {
			// Given
			customPath := generate.GrantedConfigPath + ".test"

			// When
			err := generate.CreateGrantedConfiguration(customPath)

			// Then
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(customPath).To(gomega.BeARegularFile())
		})
	})
})
