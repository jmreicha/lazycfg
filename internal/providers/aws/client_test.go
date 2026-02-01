package aws

import (
	"context"
	"testing"
)

func TestNewSSOClient(t *testing.T) {
	_, err := newSSOClient(context.Background(), "us-east-1", "token")
	if err != nil {
		t.Fatalf("newSSOClient failed: %v", err)
	}
}
