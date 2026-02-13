package aws

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

const (
	tokenTestRegion = "us-east-1"
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

	startURL := "https://example.awsapps.com/start"
	awsToken := `{"accessToken":"old","expiresAt":"2026-01-01T13:00:00Z","issuedAt":"2026-01-01T10:00:00Z","region":"` + tokenTestRegion + `","startUrl":"` + startURL + `"}`
	if err := os.WriteFile(filepath.Join(awsCache, "token.json"), []byte(awsToken), 0600); err != nil {
		t.Fatalf("write aws token: %v", err)
	}

	grantedToken := `{"accessToken":"new","expiresAt":"2026-01-01T14:00:00Z","issuedAt":"2026-01-01T11:00:00Z","region":"` + tokenTestRegion + `","startUrl":"` + startURL + `"}`
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
	startURL := "https://example.awsapps.com/start"
	token := `{"accessToken":"expired","expiresAt":"2026-01-01T11:00:00Z","issuedAt":"2026-01-01T10:00:00Z","region":"` + tokenTestRegion + `","startUrl":"` + startURL + `"}`
	if err := os.WriteFile(filepath.Join(cache, "token.json"), []byte(token), 0600); err != nil {
		t.Fatalf("write token: %v", err)
	}

	_, err := LoadNewestToken([]string{cache}, now)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestLoadNewestTokenEmptyPaths(t *testing.T) {
	_, err := LoadNewestToken(nil, time.Now())
	if err == nil {
		t.Fatal("expected error for empty cache paths")
	}
}

func TestLoadNewestTokenInvalidTokenFile(t *testing.T) {
	cache := t.TempDir()
	if err := os.WriteFile(filepath.Join(cache, "token.json"), []byte("not-json"), 0600); err != nil {
		t.Fatalf("write token: %v", err)
	}

	_, err := LoadNewestToken([]string{cache}, time.Now())
	if err == nil {
		t.Fatal("expected error for invalid token file")
	}
}

func TestLoadNewestTokenMissingIssuedAt(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	cache := t.TempDir()
	token := `{"accessToken":"valid","expiresAt":"2026-01-01T14:00:00Z","region":"` + tokenTestRegion + `","startUrl":"https://example.awsapps.com/start"}`
	if err := os.WriteFile(filepath.Join(cache, "token.json"), []byte(token), 0600); err != nil {
		t.Fatalf("write token: %v", err)
	}

	selected, err := LoadNewestToken([]string{cache}, now)
	if err != nil {
		t.Fatalf("LoadNewestToken failed: %v", err)
	}
	if selected.AccessToken != "valid" {
		t.Fatalf("selected token = %q, want %q", selected.AccessToken, "valid")
	}
	if !selected.IssuedAt.IsZero() {
		t.Fatalf("expected zero IssuedAt, got %v", selected.IssuedAt)
	}
}

func TestLoadMatchingTokenFiltersSession(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	cache := t.TempDir()
	other := `{"accessToken":"other","expiresAt":"2026-01-01T15:00:00Z","issuedAt":"2026-01-01T10:00:00Z","region":"eu-west-1","startUrl":"https://other.awsapps.com/start"}`
	matching := `{"accessToken":"match","expiresAt":"2026-01-01T14:00:00Z","issuedAt":"2026-01-01T10:00:00Z","region":"` + tokenTestRegion + `","startUrl":"https://example.awsapps.com/start"}`
	if err := os.WriteFile(filepath.Join(cache, "other.json"), []byte(other), 0600); err != nil {
		t.Fatalf("write other token: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cache, "match.json"), []byte(matching), 0600); err != nil {
		t.Fatalf("write matching token: %v", err)
	}

	selected, err := LoadMatchingToken([]string{cache}, "https://example.awsapps.com/start", tokenTestRegion, now)
	if err != nil {
		t.Fatalf("LoadMatchingToken failed: %v", err)
	}
	if selected.AccessToken != "match" {
		t.Fatalf("selected token = %q, want %q", selected.AccessToken, "match")
	}
}

func TestLoadTokensFromMissingPath(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")

	tokens, err := loadTokensFromMissingPath(missing)
	if err != nil {
		t.Fatalf("loadTokensFromMissingPath failed: %v", err)
	}
	if len(tokens) != 0 {
		t.Fatalf("expected no tokens for missing path")
	}
}
