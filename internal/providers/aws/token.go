package aws

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const tokenExpiryLayout = "2006-01-02T15:04:05Z"

var errNoValidToken = errors.New("no valid sso token found")

// LoadNewestToken finds the newest valid SSO token across cache paths.
func LoadNewestToken(cachePaths []string, now time.Time) (SSOToken, error) {
	tokens, err := loadTokens(cachePaths)
	if err != nil {
		return SSOToken{}, err
	}

	valid := filterValidTokens(tokens, now)
	if len(valid) == 0 {
		return SSOToken{}, errNoValidToken
	}

	sort.Slice(valid, func(i, j int) bool {
		if valid[i].ExpiresAt.Equal(valid[j].ExpiresAt) {
			return valid[i].IssuedAt.After(valid[j].IssuedAt)
		}
		return valid[i].ExpiresAt.After(valid[j].ExpiresAt)
	})

	return valid[0], nil
}

func loadTokens(cachePaths []string) ([]SSOToken, error) {
	if len(cachePaths) == 0 {
		return nil, errTokenCachePathsEmpty
	}

	tokens := []SSOToken{}
	for _, path := range cachePaths {
		entries, err := loadTokensFromPath(path)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, entries...)
	}

	return tokens, nil
}

func loadTokensFromPath(path string) ([]SSOToken, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errTokenCachePathsEmpty
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read token cache directory %q: %w", path, err)
	}

	tokens := []SSOToken{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		token, err := readToken(filepath.Join(path, entry.Name()))
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}

	return tokens, nil
}

func readToken(path string) (SSOToken, error) {
	// #nosec G304 -- token cache path is user configured.
	file, err := os.Open(path)
	if err != nil {
		return SSOToken{}, fmt.Errorf("open token file %q: %w", path, err)
	}
	defer func() {
		_ = file.Close()
	}()

	data, err := io.ReadAll(file)
	if err != nil {
		return SSOToken{}, fmt.Errorf("read token file %q: %w", path, err)
	}

	var raw tokenFile
	if err := json.Unmarshal(data, &raw); err != nil {
		return SSOToken{}, fmt.Errorf("parse token file %q: %w", path, err)
	}

	token, err := raw.toToken()
	if err != nil {
		return SSOToken{}, fmt.Errorf("parse token fields %q: %w", path, err)
	}

	return token, nil
}

func filterValidTokens(tokens []SSOToken, now time.Time) []SSOToken {
	valid := make([]SSOToken, 0, len(tokens))
	for _, token := range tokens {
		if token.IsExpired(now) {
			continue
		}
		valid = append(valid, token)
	}

	return valid
}

type tokenFile struct {
	AccessToken string `json:"accessToken"`
	ExpiresAt   string `json:"expiresAt"`
	IssuedAt    string `json:"issuedAt"`
	Region      string `json:"region"`
	StartURL    string `json:"startUrl"`
}

func (t tokenFile) toToken() (SSOToken, error) {
	if strings.TrimSpace(t.AccessToken) == "" {
		return SSOToken{}, errors.New("access token missing")
	}
	expiresAt, err := time.Parse(tokenExpiryLayout, t.ExpiresAt)
	if err != nil {
		return SSOToken{}, fmt.Errorf("parse expiresAt: %w", err)
	}
	issuedAt, err := time.Parse(tokenExpiryLayout, t.IssuedAt)
	if err != nil {
		return SSOToken{}, fmt.Errorf("parse issuedAt: %w", err)
	}

	return SSOToken{
		AccessToken: t.AccessToken,
		ExpiresAt:   expiresAt,
		IssuedAt:    issuedAt,
		Region:      t.Region,
		StartURL:    t.StartURL,
	}, nil
}

// SSOToken represents an SSO access token payload.
type SSOToken struct {
	AccessToken string
	ExpiresAt   time.Time
	IssuedAt    time.Time
	Region      string
	StartURL    string
}

// IsExpired reports whether the token is expired at the provided time.
func (t SSOToken) IsExpired(now time.Time) bool {
	return !t.ExpiresAt.After(now)
}

// MatchesSession reports whether the token matches the provided session metadata.
func (t SSOToken) MatchesSession(startURL, region string) bool {
	return strings.EqualFold(strings.TrimSpace(t.StartURL), strings.TrimSpace(startURL)) &&
		strings.EqualFold(strings.TrimSpace(t.Region), strings.TrimSpace(region))
}
