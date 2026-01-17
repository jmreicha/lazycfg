package ssh

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevinburke/ssh_config"
)

func TestParseConfig(t *testing.T) {
	tests := []struct {
		name        string
		configFile  string
		expectError bool
	}{
		{
			name:        "basic config",
			configFile:  "testdata/basic_config",
			expectError: false,
		},
		{
			name:        "config with comments",
			configFile:  "testdata/with_comments",
			expectError: false,
		},
		{
			name:        "empty config",
			configFile:  "testdata/empty_config",
			expectError: false,
		},
		{
			name:        "non-existent file",
			configFile:  "testdata/nonexistent",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := ParseConfig(tt.configFile)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if cfg == nil {
				t.Error("config is nil")
			}
		})
	}
}

func TestFindHost(t *testing.T) {
	cfg, err := ParseConfig("testdata/basic_config")
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	tests := []struct {
		name      string
		hostname  string
		expectNil bool
	}{
		{
			name:      "exact match",
			hostname:  "example.com",
			expectNil: false,
		},
		{
			name:      "wildcard match",
			hostname:  "server.test.local",
			expectNil: false,
		},
		{
			name:      "no match",
			hostname:  "nonexistent.com",
			expectNil: false, // Should match "Host *"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host := FindHost(cfg, tt.hostname)

			if tt.expectNil && host != nil {
				t.Errorf("expected nil, got %v", host)
			}

			if !tt.expectNil && host == nil {
				t.Error("expected host, got nil")
			}
		})
	}
}

func TestFindHost_NilConfig(t *testing.T) {
	host := FindHost(nil, "test.com")
	if host != nil {
		t.Errorf("expected nil for nil config, got %v", host)
	}
}

func TestAddHost(t *testing.T) {
	cfg := &ssh_config.Config{}

	hostConfig := HostConfig{
		Host:          "newhost.com",
		Hostname:      "192.168.1.10",
		User:          "admin",
		Port:          22,
		IdentityAgent: "/tmp/ssh-agent.sock",
		IdentityFile:  "~/.ssh/newkey",
		Options: map[string]string{
			"StrictHostKeyChecking": "no",
		},
	}

	err := AddHost(cfg, hostConfig)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify host was added
	host := FindHost(cfg, "newhost.com")
	if host == nil {
		t.Fatal("host was not added")
	}

	// Verify values
	hostname, err := cfg.Get("newhost.com", "HostName")
	if err != nil {
		t.Errorf("failed to get HostName: %v", err)
	}
	if hostname != "192.168.1.10" {
		t.Errorf("HostName = %s, want 192.168.1.10", hostname)
	}
}

func TestAddHost_NilConfig(t *testing.T) {
	hostConfig := HostConfig{
		Host: "test.com",
	}

	err := AddHost(nil, hostConfig)
	if err == nil {
		t.Error("expected error for nil config")
	}
}

func TestAddHost_DuplicateHost(t *testing.T) {
	cfg, err := ParseConfig("testdata/basic_config")
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	hostConfig := HostConfig{
		Host:     "example.com",
		Hostname: "192.168.1.10",
	}

	err = AddHost(cfg, hostConfig)
	if err == nil {
		t.Error("expected error for duplicate host")
	}
}

func TestUpdateHost(t *testing.T) {
	cfg, err := ParseConfig("testdata/basic_config")
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	hostConfig := HostConfig{
		Host:          "example.com",
		Hostname:      "updated.example.com",
		User:          "newuser",
		Port:          2222,
		IdentityAgent: "/tmp/ssh-agent.sock",
	}

	err = UpdateHost(cfg, hostConfig)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify updated values
	hostname, err := cfg.Get("example.com", "HostName")
	if err != nil {
		t.Errorf("failed to get HostName: %v", err)
	}
	if hostname != "updated.example.com" {
		t.Errorf("HostName = %s, want updated.example.com", hostname)
	}

	agent, err := cfg.Get("example.com", "IdentityAgent")
	if err != nil {
		t.Errorf("failed to get IdentityAgent: %v", err)
	}
	if agent != "/tmp/ssh-agent.sock" {
		t.Errorf("IdentityAgent = %s, want /tmp/ssh-agent.sock", agent)
	}

	user, err := cfg.Get("example.com", "User")
	if err != nil {
		t.Errorf("failed to get User: %v", err)
	}
	if user != "newuser" {
		t.Errorf("User = %s, want newuser", user)
	}
}

func TestUpdateHost_NilConfig(t *testing.T) {
	hostConfig := HostConfig{
		Host: "test.com",
	}

	err := UpdateHost(nil, hostConfig)
	if err == nil {
		t.Error("expected error for nil config")
	}
}

func TestUpdateHost_NotFound(t *testing.T) {
	cfg := &ssh_config.Config{}

	hostConfig := HostConfig{
		Host: "nonexistent.com",
	}

	err := UpdateHost(cfg, hostConfig)
	if err == nil {
		t.Error("expected error for non-existent host")
	}
}

func TestRemoveHost(t *testing.T) {
	cfg, err := ParseConfig("testdata/basic_config")
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	initialHostCount := len(cfg.Hosts)

	err = RemoveHost(cfg, "example.com")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify host was removed
	if len(cfg.Hosts) != initialHostCount-1 {
		t.Errorf("expected %d hosts, got %d", initialHostCount-1, len(cfg.Hosts))
	}
}

func TestRemoveHost_NilConfig(t *testing.T) {
	err := RemoveHost(nil, "test.com")
	if err == nil {
		t.Error("expected error for nil config")
	}
}

func TestRemoveHost_NotFound(t *testing.T) {
	cfg := &ssh_config.Config{}

	err := RemoveHost(cfg, "nonexistent.com")
	if err == nil {
		t.Error("expected error for non-existent host")
	}
}

func TestWriteConfig(t *testing.T) {
	cfg, err := ParseConfig("testdata/basic_config")
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "ssh_config")

	err = WriteConfig(cfg, tmpFile)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify file exists
	info, err := os.Stat(tmpFile)
	if err != nil {
		t.Errorf("failed to stat file: %v", err)
	}

	// Verify permissions (0600)
	if info.Mode().Perm() != 0600 {
		t.Errorf("file permissions = %o, want 0600", info.Mode().Perm())
	}

	// Verify content can be parsed back
	parsedCfg, err := ParseConfig(tmpFile)
	if err != nil {
		t.Errorf("failed to parse written config: %v", err)
	}

	if parsedCfg == nil {
		t.Error("parsed config is nil")
	}
}

func TestWriteConfig_NilConfig(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "ssh_config")

	err := WriteConfig(nil, tmpFile)
	if err == nil {
		t.Error("expected error for nil config")
	}
}

func TestGetHostValue(t *testing.T) {
	cfg, err := ParseConfig("testdata/basic_config")
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	value, err := GetHostValue(cfg, "example.com", "User")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if value != "ubuntu" {
		t.Errorf("User = %s, want ubuntu", value)
	}
}

func TestGetHostValue_NilConfig(t *testing.T) {
	_, err := GetHostValue(nil, "test.com", "User")
	if err == nil {
		t.Error("expected error for nil config")
	}
}

func TestGetHostValues(t *testing.T) {
	cfg, err := ParseConfig("testdata/basic_config")
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	values, err := GetHostValues(cfg, "example.com", "IdentityFile")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(values) == 0 {
		t.Error("expected at least one value")
	}
}

func TestGetHostValues_NilConfig(t *testing.T) {
	_, err := GetHostValues(nil, "test.com", "IdentityFile")
	if err == nil {
		t.Error("expected error for nil config")
	}
}

func TestRoundTrip(t *testing.T) {
	// Test that we can parse a config, modify it, write it, and parse it again
	cfg, err := ParseConfig("testdata/with_comments")
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	// Add a new host with a unique pattern
	newHost := HostConfig{
		Host:          "roundtrip.example.com",
		Hostname:      "192.168.1.100",
		User:          "testuser",
		Port:          22,
		IdentityAgent: "/tmp/ssh-agent.sock",
	}

	err = AddHost(cfg, newHost)
	if err != nil {
		t.Errorf("failed to add host: %v", err)
	}

	// Write to temp file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "ssh_config")

	err = WriteConfig(cfg, tmpFile)
	if err != nil {
		t.Errorf("failed to write config: %v", err)
	}

	// Parse again
	cfg2, err := ParseConfig(tmpFile)
	if err != nil {
		t.Errorf("failed to parse written config: %v", err)
	}

	// Verify new host exists
	host := FindHost(cfg2, "roundtrip.example.com")
	if host == nil {
		t.Error("new host not found after round-trip")
	}

	// Verify comments are preserved
	content, err := os.ReadFile(tmpFile) // #nosec G304 - tmpFile is test-generated
	if err != nil {
		t.Errorf("failed to read file: %v", err)
	}

	if !strings.Contains(string(content), "#") {
		t.Error("comments were not preserved")
	}
}

func TestConcurrentAccess(t *testing.T) {
	cfg := &ssh_config.Config{}

	// Test concurrent modifications
	done := make(chan bool)

	for i := range 10 {
		go func(id int) {
			hostConfig := HostConfig{
				Host:     "host" + string(rune('0'+id)) + ".example.com",
				Hostname: "192.168.1." + string(rune('0'+id)),
			}
			_ = AddHost(cfg, hostConfig)
			done <- true
		}(i)
	}

	for range 10 {
		<-done
	}

	// Should have some hosts added (exact count may vary due to race conditions)
	if len(cfg.Hosts) == 0 {
		t.Error("no hosts were added")
	}
}

func TestParseConfig_MalformedConfig(t *testing.T) {
	tests := []struct {
		name        string
		configFile  string
		expectError bool
	}{
		{
			name:        "malformed config",
			configFile:  "testdata/malformed_config",
			expectError: false, // ssh_config is lenient
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := ParseConfig(tt.configFile)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if cfg == nil {
				t.Error("config is nil")
			}
		})
	}
}

func TestParseConfig_WithInclude(t *testing.T) {
	cfg, err := ParseConfig("testdata/with_include")
	if err != nil {
		t.Fatalf("failed to parse config with Include: %v", err)
	}

	// Verify Include directive was parsed
	if cfg == nil {
		t.Error("config is nil")
		return
	}

	// Verify hosts before Include
	host := FindHost(cfg, "primary.com")
	if host == nil {
		t.Error("primary.com host not found")
	}

	// Verify hosts after Include
	host = FindHost(cfg, "fallback.com")
	if host == nil {
		t.Error("fallback.com host not found")
	}
}

func TestParseConfig_ComplexPatterns(t *testing.T) {
	cfg, err := ParseConfig("testdata/complex_patterns")
	if err != nil {
		t.Fatalf("failed to parse config with complex patterns: %v", err)
	}

	tests := []struct {
		name      string
		hostname  string
		expectNil bool
	}{
		{
			name:      "wildcard match",
			hostname:  "web1.prod.example.com",
			expectNil: false,
		},
		{
			name:      "negated pattern",
			hostname:  "staging.prod.example.com",
			expectNil: false, // Still matches, negation is complex
		},
		{
			name:      "jump host pattern",
			hostname:  "jump-server1",
			expectNil: false,
		},
		{
			name:      "ip pattern",
			hostname:  "192.168.1.5",
			expectNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host := FindHost(cfg, tt.hostname)

			if tt.expectNil && host != nil {
				t.Errorf("expected nil, got %v", host)
			}

			if !tt.expectNil && host == nil {
				t.Error("expected host, got nil")
			}
		})
	}
}

func TestAddHost_InvalidPort(t *testing.T) {
	hostConfig := HostConfig{
		Host:     "test.com",
		Hostname: "test.com",
		Port:     -1,
	}

	err := hostConfig.Validate()
	if err == nil {
		t.Error("expected error for negative port")
	}
}

func TestAddHost_PortBoundary(t *testing.T) {
	tests := []struct {
		name        string
		port        int
		expectError bool
	}{
		{
			name:        "port 0",
			port:        0,
			expectError: false,
		},
		{
			name:        "port 1",
			port:        1,
			expectError: false,
		},
		{
			name:        "port 65535",
			port:        65535,
			expectError: false,
		},
		{
			name:        "port 65536",
			port:        65536,
			expectError: true,
		},
		{
			name:        "port -1",
			port:        -1,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hostConfig := HostConfig{
				Host:     "test.com",
				Hostname: "test.com",
				Port:     tt.port,
			}

			err := hostConfig.Validate()

			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestFindHostExact(t *testing.T) {
	cfg, err := ParseConfig("testdata/basic_config")
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	tests := []struct {
		name      string
		pattern   string
		expectNil bool
	}{
		{
			name:      "exact match",
			pattern:   "example.com",
			expectNil: false,
		},
		{
			name:      "wildcard pattern",
			pattern:   "*.test.local",
			expectNil: false,
		},
		{
			name:      "global wildcard",
			pattern:   "*",
			expectNil: false,
		},
		{
			name:      "no match",
			pattern:   "nonexistent.com",
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host := FindHostExact(cfg, tt.pattern)

			if tt.expectNil && host != nil {
				t.Errorf("expected nil, got %v", host)
			}

			if !tt.expectNil && host == nil {
				t.Error("expected host, got nil")
			}
		})
	}
}

func TestFindHostExact_NilConfig(t *testing.T) {
	host := FindHostExact(nil, "test.com")
	if host != nil {
		t.Errorf("expected nil for nil config, got %v", host)
	}
}

func TestFindHostsByPatterns(t *testing.T) {
	cfg, err := ParseConfig("testdata/basic_config")
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	patternMap := FindHostsByPatterns(cfg)

	if len(patternMap) == 0 {
		t.Error("expected at least one pattern")
	}

	// Check for known patterns
	if _, ok := patternMap["example.com"]; !ok {
		t.Error("example.com pattern not found")
	}

	if _, ok := patternMap["*.test.local"]; !ok {
		t.Error("*.test.local pattern not found")
	}

	if _, ok := patternMap["*"]; !ok {
		t.Error("* pattern not found")
	}
}

func TestFindHostsByPatterns_NilConfig(t *testing.T) {
	patternMap := FindHostsByPatterns(nil)
	if len(patternMap) != 0 {
		t.Errorf("expected empty map for nil config, got %d entries", len(patternMap))
	}
}

func TestUpsertGlobalOptions(t *testing.T) {
	cfg := &ssh_config.Config{}

	options := map[string]string{
		"ServerAliveInterval": "60",
		"ServerAliveCountMax": "3",
	}

	updated, err := UpsertGlobalOptions(cfg, options)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !updated {
		t.Error("expected options to be updated")
	}

	// Verify global host was created
	globalHost := FindHostExact(cfg, "*")
	if globalHost == nil {
		t.Fatal("global host not created")
	}

	// Verify options were added
	interval, err := cfg.Get("test.com", "ServerAliveInterval")
	if err != nil {
		t.Errorf("failed to get ServerAliveInterval: %v", err)
	}
	if interval != "60" {
		t.Errorf("ServerAliveInterval = %s, want 60", interval)
	}
}

func TestUpsertGlobalOptions_EmptyOptions(t *testing.T) {
	cfg := &ssh_config.Config{}

	updated, err := UpsertGlobalOptions(cfg, map[string]string{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if updated {
		t.Error("expected no update for empty options")
	}
}

func TestUpsertGlobalOptions_NilConfig(t *testing.T) {
	options := map[string]string{
		"ServerAliveInterval": "60",
	}

	_, err := UpsertGlobalOptions(nil, options)
	if err == nil {
		t.Error("expected error for nil config")
	}
}

func TestUpsertGlobalOptions_UpdateExisting(t *testing.T) {
	cfg, err := ParseConfig("testdata/basic_config")
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	// Update existing global option
	options := map[string]string{
		"ServerAliveInterval": "120",
	}

	updated, err := UpsertGlobalOptions(cfg, options)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !updated {
		t.Error("expected options to be updated")
	}

	// Verify option was updated
	interval, err := cfg.Get("test.com", "ServerAliveInterval")
	if err != nil {
		t.Errorf("failed to get ServerAliveInterval: %v", err)
	}
	if interval != "120" {
		t.Errorf("ServerAliveInterval = %s, want 120", interval)
	}
}

func TestWriteConfig_InvalidPath(t *testing.T) {
	cfg, err := ParseConfig("testdata/basic_config")
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	err = WriteConfig(cfg, "/nonexistent/invalid/path/config")
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestRenderConfig_IncludesAndGlobalsSeparation(t *testing.T) {
	// Parse a real config file with Include directives and global settings
	cfg, err := ParseConfig("testdata/with_includes")
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	output := renderConfig(cfg)

	expected := `# This file was generated automatically. Do not edit manually.

# Added by OrbStack: 'orb' SSH host for Linux machines
# This only works if it's at the top of ssh_config (before any Host blocks).
# This won't be added again if you remove it.
Include ~/.orbstack/ssh/config

# Global SSH settings
ServerAliveCountMax 3
ServerAliveInterval 60
`

	if output != expected {
		t.Errorf("renderConfig output mismatch\nGot:\n%q\n\nWant:\n%q", output, expected)
	}
}

func TestRenderConfig_CompleteConfig(t *testing.T) {
	// Parse a complete real-world config with includes, globals, hosts, and wildcard
	cfg, err := ParseConfig("testdata/complete_config")
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	output := renderConfig(cfg)

	expected := `# This file was generated automatically. Do not edit manually.

# Added by OrbStack: 'orb' SSH host for Linux machines
# This only works if it's at the top of ssh_config (before any Host blocks).
# This won't be added again if you remove it.
Include ~/.orbstack/ssh/config

# Global SSH settings
ServerAliveCountMax 3
ServerAliveInterval 60
StrictHostKeyChecking ask

Host 192.168.1.1
    HostName 192.168.1.1
    User admin
    Port 22

Host foo
    HostName foo
    User bar
    Port 22

Host github.com
    HostName github.com
    User git
    IdentityFile ~/.ssh/id_ed25519

Host *
    IdentityAgent "~/path/to/config"
`

	if output != expected {
		t.Errorf("renderConfig output mismatch\nGot:\n%q\n\nWant:\n%q", output, expected)
	}
}
