package generate_test

import (
	"os"
	"strings"
	"testing"

	"github.com/jmreicha/lazycfg/internal/cmd/generate"
	"github.com/lithammer/dedent"
	"github.com/stretchr/testify/assert"
)

// teardown cleans up the environment after tests.
func teardown() {
	os.Remove(generate.GrantedConfigPath + ".test")
}

// TestCreateGrantedConfiguration tests the creation of the Granted configuration file.
func TestCreateGrantedConfiguration(t *testing.T) {
	defer teardown()

	configPath := generate.GrantedConfigPath
	err := generate.CreateGrantedConfiguration(configPath)

	assert.NoError(t, err)
	assert.FileExists(t, configPath)

	// Verify the content of the file
	content, readErr := os.ReadFile(configPath)
	assert.NoError(t, readErr)
	expectedContent := dedent.Dedent(`
		DefaultBrowser = "STDOUT"
		CustomBrowserPath = ""
		CustomSSOBrowserPath = ""
		Ordering = ""
		ExportCredentialSuffix = ""
		DisableUsageTips = true
		CredentialProcessAutoLogin = true
	`)
	expectedContent = strings.TrimSpace(expectedContent)

	assert.Equal(t, expectedContent, string(content))
}

// TestCreateGrantedConfigurationDefaultLocation verifies that the default config file is created.
func TestCreateGrantedConfigurationDefaultLocation(t *testing.T) {
	defer os.Remove(generate.GrantedConfigPath + ".test")

	configPath := generate.GrantedConfigPath + ".test"

	err := generate.CreateGrantedConfiguration(configPath)

	assert.NoError(t, err)
	assert.FileExists(t, configPath)
}
