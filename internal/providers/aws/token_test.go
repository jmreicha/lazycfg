package aws

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadNewestToken(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	base := t.TempDir()
	awsCache := filepath.Join(base, "aws")
	grantedCache := filepath.Join(base, "granted")
	if err := os.MkdirAll(awsCache, 0700); err != nil {
		t.Fatalf("mkdir aws cache: %v", err)
	}
	if err := os.MkdirAll(grantedCache, 0700); err != nil {
		t.Fatalf("mkdir granted cache: %v", err)
	}

	awsToken := `{"accessToken":"old","expiresAt":"2026-01-01T13:00:00Z","issuedAt":"2026-01-01T10:00:00Z","region":"us-east-1","startUrl":"https://example.awsapps.com/start"}`
	if err := os.WriteFile(filepath.Join(awsCache, "token.json"), []byte(awsToken), 0600); err != nil {
		t.Fatalf("write aws token: %v", err)
	}

	grantedToken := `{"accessToken":"new","expiresAt":"2026-01-01T14:00:00Z","issuedAt":"2026-01-01T11:00:00Z","region":"us-east-1","startUrl":"https://example.awsapps.com/start"}`
	if err := os.WriteFile(filepath.Join(grantedCache, "token.json"), []byte(grantedToken), 0600); err != nil {
		t.Fatalf("write granted token: %v", err)
	}

	selected, err := LoadNewestToken([]string{awsCache, grantedCache}, now)
	if err != nil {
		t.Fatalf("LoadNewestToken failed: %v", err)
	}

	if selected.AccessToken != "new" {
		t.Fatalf("selected token = %q", selected.AccessToken)
	}
}

func TestLoadNewestTokenExpired(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	cache := t.TempDir()
	token := `{"accessToken":"expired","expiresAt":"2026-01-01T11:00:00Z","issuedAt":"2026-01-01T10:00:00Z","region":"us-east-1","startUrl":"https://example.awsapps.com/start"}`
	if err := os.WriteFile(filepath.Join(cache, "token.json"), []byte(token), 0600); err != nil {
		t.Fatalf("write token: %v", err)
	}

	_, err := LoadNewestToken([]string{cache}, now)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}
