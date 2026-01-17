package generate

import (
	"runtime"
	"testing"
)

func TestGetOSTemplateGranted(t *testing.T) {
	template := getOSTemplateGranted()

	switch runtime.GOOS {
	case "darwin", "linux":
		if template == "" {
			t.Fatal("expected template content")
		}
	default:
		if template != "" {
			t.Fatalf("expected empty template for %s", runtime.GOOS)
		}
	}
}
