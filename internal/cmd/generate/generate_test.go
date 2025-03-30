package generate_test

import (
	"os"
	"runtime"
	"testing"

	"github.com/jmreicha/lazycfg/internal/cmd/generate"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGenerate(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Generate Suite")
}

var _ = Describe("Generate", func() {
	AfterEach(func() {
		// Clean up generated files after each test
		os.Remove(generate.GrantedConfigPath)
		os.Remove(generate.GrantedConfigPath + ".test")
	})

	Describe("CreateGrantedConfiguration", func() {
		It("should create a granted config file with OS-specific content", func() {
			// When
			err := generate.CreateGrantedConfiguration(generate.GrantedConfigPath)

			// Then
			Expect(err).NotTo(HaveOccurred())
			Expect(generate.GrantedConfigPath).To(BeARegularFile())

			// Verify the content of the file
			content, err := os.ReadFile(generate.GrantedConfigPath)
			Expect(err).NotTo(HaveOccurred())

			// Common configuration should be present regardless of OS
			Expect(string(content)).To(ContainSubstring(`DefaultBrowser = "STDOUT"`))
			Expect(string(content)).To(ContainSubstring(`DisableUsageTips = true`))

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

		It("should create a config file at a custom location", func() {
			// Given
			customPath := generate.GrantedConfigPath + ".test"

			// When
			err := generate.CreateGrantedConfiguration(customPath)

			// Then
			Expect(err).NotTo(HaveOccurred())
			Expect(customPath).To(BeARegularFile())
		})
	})
})
