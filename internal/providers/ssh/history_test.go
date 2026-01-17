package ssh

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSSHCommand(t *testing.T) {
	tests := []struct {
		name   string
		line   string
		want   Command
		wantOK bool
	}{
		{
			name: "simple hostname",
			line: "ssh example.com",
			want: Command{
				Host:     "example.com",
				Hostname: "example.com",
				Port:     22,
			},
			wantOK: true,
		},
		{
			name: "user@hostname",
			line: "ssh user@example.com",
			want: Command{
				Host:     "example.com",
				Hostname: "example.com",
				User:     "user",
				Port:     22,
			},
			wantOK: true,
		},
		{
			name: "with port flag",
			line: "ssh -p 2222 user@example.com",
			want: Command{
				Host:     "example.com",
				Hostname: "example.com",
				User:     "user",
				Port:     2222,
			},
			wantOK: true,
		},
		{
			name: "with identity file",
			line: "ssh -i ~/.ssh/id_rsa user@example.com",
			want: Command{
				Host:         "example.com",
				Hostname:     "example.com",
				User:         "user",
				Port:         22,
				IdentityFile: homeDir(t) + "/.ssh/id_rsa",
			},
			wantOK: true,
		},
		{
			name: "complex command",
			line: "ssh -i ~/.ssh/key -p 2222 deploy@prod.example.com",
			want: Command{
				Host:         "prod.example.com",
				Hostname:     "prod.example.com",
				User:         "deploy",
				Port:         2222,
				IdentityFile: homeDir(t) + "/.ssh/key",
			},
			wantOK: true,
		},
		{
			name: "with command after pipe",
			line: "echo test | ssh user@example.com",
			want: Command{
				Host:     "example.com",
				Hostname: "example.com",
				User:     "user",
				Port:     22,
			},
			wantOK: true,
		},
		{
			name:   "not ssh command",
			line:   "ls -la",
			wantOK: false,
		},
		{
			name:   "empty line",
			line:   "",
			wantOK: false,
		},
		{
			name:   "ssh with verbose flag only",
			line:   "ssh -v",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseSSHCommand(tt.line)
			if ok != tt.wantOK {
				t.Errorf("parseSSHCommand() ok = %v, want %v", ok, tt.wantOK)
				return
			}
			if !tt.wantOK {
				return
			}
			if got.Host != tt.want.Host {
				t.Errorf("Host = %q, want %q", got.Host, tt.want.Host)
			}
			if got.Hostname != tt.want.Hostname {
				t.Errorf("Hostname = %q, want %q", got.Hostname, tt.want.Hostname)
			}
			if got.User != tt.want.User {
				t.Errorf("User = %q, want %q", got.User, tt.want.User)
			}
			if got.Port != tt.want.Port {
				t.Errorf("Port = %d, want %d", got.Port, tt.want.Port)
			}
			if got.IdentityFile != tt.want.IdentityFile {
				t.Errorf("IdentityFile = %q, want %q", got.IdentityFile, tt.want.IdentityFile)
			}
		})
	}
}

func TestParseHistoryFile(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name      string
		content   string
		wantCount int
		wantFirst Command
	}{
		{
			name: "bash history",
			content: `ls -la
ssh user@example.com
cd /tmp
ssh admin@server.com -p 2222
`,
			wantCount: 2,
			wantFirst: Command{
				Host:     "example.com",
				Hostname: "example.com",
				User:     "user",
				Port:     22,
			},
		},
		{
			name: "zsh extended history",
			content: `: 1704067200:0;ssh user@example.com
: 1704067300:0;ls -la
: 1704067400:0;ssh admin@server.com
`,
			wantCount: 2,
			wantFirst: Command{
				Host:     "example.com",
				Hostname: "example.com",
				User:     "user",
				Port:     22,
			},
		},
		{
			name:      "empty file",
			content:   "",
			wantCount: 0,
		},
		{
			name: "no ssh commands",
			content: `ls -la
cd /tmp
echo hello
`,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			histFile := filepath.Join(tmpDir, "history")
			if err := os.WriteFile(histFile, []byte(tt.content), 0600); err != nil {
				t.Fatalf("failed to write test history file: %v", err)
			}

			got, err := parseHistoryFile(histFile)
			if err != nil {
				t.Errorf("parseHistoryFile() error = %v", err)
				return
			}

			if len(got) != tt.wantCount {
				t.Errorf("parseHistoryFile() count = %d, want %d", len(got), tt.wantCount)
				return
			}

			if tt.wantCount > 0 {
				first := got[0]
				if first.Host != tt.wantFirst.Host {
					t.Errorf("first Host = %q, want %q", first.Host, tt.wantFirst.Host)
				}
				if first.User != tt.wantFirst.User {
					t.Errorf("first User = %q, want %q", first.User, tt.wantFirst.User)
				}
				if first.Port != tt.wantFirst.Port {
					t.Errorf("first Port = %d, want %d", first.Port, tt.wantFirst.Port)
				}
			}
		})
	}
}

func TestParseHistoryFiles(t *testing.T) {
	// This test is less deterministic as it depends on actual shell history files
	// We'll just verify it doesn't crash and returns a reasonable result
	commands, err := ParseHistoryFiles()
	if err != nil {
		t.Logf("ParseHistoryFiles() returned error (may be expected): %v", err)
	}

	// Even if no history files exist, we should get an empty slice, not an error
	if commands == nil {
		t.Error("ParseHistoryFiles() returned nil slice")
	}
}

func TestCommand_ConvertToHostConfig(t *testing.T) {
	tests := []struct {
		name string
		cmd  Command
		want HostConfig
	}{
		{
			name: "basic command",
			cmd: Command{
				Host:     "example.com",
				Hostname: "example.com",
				Port:     22,
				User:     "user",
			},
			want: HostConfig{
				Host:     "example.com",
				Hostname: "example.com",
				Port:     22,
				User:     "user",
			},
		},
		{
			name: "with identity file",
			cmd: Command{
				Host:         "example.com",
				Hostname:     "example.com",
				Port:         2222,
				User:         "admin",
				IdentityFile: "/home/user/.ssh/id_rsa",
			},
			want: HostConfig{
				Host:         "example.com",
				Hostname:     "example.com",
				Port:         2222,
				User:         "admin",
				IdentityFile: "/home/user/.ssh/id_rsa",
			},
		},
		{
			name: "minimal command",
			cmd: Command{
				Host:     "server",
				Hostname: "server",
				Port:     22,
			},
			want: HostConfig{
				Host:     "server",
				Hostname: "server",
				Port:     22,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cmd.ConvertToHostConfig()
			if got.Host != tt.want.Host {
				t.Errorf("Host = %q, want %q", got.Host, tt.want.Host)
			}
			if got.Hostname != tt.want.Hostname {
				t.Errorf("Hostname = %q, want %q", got.Hostname, tt.want.Hostname)
			}
			if got.Port != tt.want.Port {
				t.Errorf("Port = %d, want %d", got.Port, tt.want.Port)
			}
			if got.User != tt.want.User {
				t.Errorf("User = %q, want %q", got.User, tt.want.User)
			}
			if got.IdentityFile != tt.want.IdentityFile {
				t.Errorf("IdentityFile = %q, want %q", got.IdentityFile, tt.want.IdentityFile)
			}
		})
	}
}

func homeDir(t *testing.T) string {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}
	return home
}
