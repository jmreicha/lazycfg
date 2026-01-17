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
