package aws

import "testing"

func TestProviderNameConstant(t *testing.T) {
	if ProviderName != "aws" {
		t.Fatalf("ProviderName = %q", ProviderName)
	}
}
