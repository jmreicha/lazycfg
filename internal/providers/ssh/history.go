package ssh

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Command represents a parsed SSH command from shell history.
type Command struct {
	Host         string
	Hostname     string
	Port         int
	User         string
	IdentityFile string
}

var (
	// sshCommandRegex matches basic SSH commands with common patterns.
	// Matches: ssh [options] [user@]hostname.
	sshCommandRegex = regexp.MustCompile(`(?:^|[;&|]\s*)ssh\s+(.+)`)

	// sshOptionsRegex matches SSH command-line options.
	portFlagRegex     = regexp.MustCompile(`-p\s+(\d+)`)
	identityFlagRegex = regexp.MustCompile(`-i\s+([^\s]+)`)
	userHostRegex     = regexp.MustCompile(`(?:^|\s)([a-zA-Z0-9_-]+)@([a-zA-Z0-9._-]+)(?:\s|$)`)
	hostOnlyRegex     = regexp.MustCompile(`(?:^|\s)([a-zA-Z0-9._-]+)(?:\s|$)`)
)

// ParseHistoryFiles reads shell history files and extracts SSH commands.
// It looks for ~/.zsh_history and ~/.bash_history by default.
func ParseHistoryFiles() ([]Command, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	historyFiles := []string{
		filepath.Join(home, ".zsh_history"),
		filepath.Join(home, ".bash_history"),
	}

	commands := make([]Command, 0)
	seen := make(map[string]bool)

	for _, histFile := range historyFiles {
		fileCommands, err := parseHistoryFile(histFile)
		if err != nil {
			// Skip files that don't exist or can't be read
			continue
		}

		// Deduplicate commands based on hostname
		for _, cmd := range fileCommands {
			key := fmt.Sprintf("%s@%s:%d", cmd.User, cmd.Hostname, cmd.Port)
			if !seen[key] {
				seen[key] = true
				commands = append(commands, cmd)
			}
		}
	}

	return commands, nil
}

// parseHistoryFile reads a single history file and extracts SSH commands.
func parseHistoryFile(path string) ([]Command, error) {
	f, err := os.Open(path) // #nosec G304 - path is user-controlled by design
	if err != nil {
		return nil, fmt.Errorf("failed to open history file: %w", err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			err = closeErr
		}
	}()

	commands := make([]Command, 0)
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()

		// Handle zsh extended history format: : timestamp:duration;command
		if strings.HasPrefix(line, ":") {
			parts := strings.SplitN(line, ";", 2)
			if len(parts) == 2 {
				line = parts[1]
			}
		}

		cmd, ok := parseSSHCommand(line)
		if ok {
			commands = append(commands, cmd)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan history file: %w", err)
	}

	return commands, nil
}

// parseSSHCommand extracts SSH connection details from a command line.
func parseSSHCommand(line string) (Command, bool) {
	// Check if line contains an SSH command
	matches := sshCommandRegex.FindStringSubmatch(line)
	if matches == nil {
		return Command{}, false
	}

	args := matches[1]
	cmd := Command{
		Port: 22, // Default SSH port
	}

	// Extract port flag (-p)
	if portMatch := portFlagRegex.FindStringSubmatch(args); portMatch != nil {
		if port, err := strconv.Atoi(portMatch[1]); err == nil {
			cmd.Port = port
		}
	}

	// Extract identity file flag (-i)
	if identityMatch := identityFlagRegex.FindStringSubmatch(args); identityMatch != nil {
		identityFile := identityMatch[1]

		// Expand tilde to home directory
		if strings.HasPrefix(identityFile, "~/") {
			home, err := os.UserHomeDir()
			if err == nil {
				identityFile = filepath.Join(home, identityFile[2:])
			}
		} else {
			// Expand environment variables
			identityFile = os.ExpandEnv(identityFile)
		}

		// Convert relative paths to absolute if possible
		if !filepath.IsAbs(identityFile) {
			home, err := os.UserHomeDir()
			if err == nil {
				identityFile = filepath.Join(home, identityFile)
			}
		}

		cmd.IdentityFile = identityFile
	}

	// Remove flags from args to extract user@host pattern
	argsWithoutFlags := portFlagRegex.ReplaceAllString(args, "")
	argsWithoutFlags = identityFlagRegex.ReplaceAllString(argsWithoutFlags, "")

	// Extract user@hostname pattern
	if userHostMatch := userHostRegex.FindStringSubmatch(argsWithoutFlags); userHostMatch != nil {
		cmd.User = userHostMatch[1]
		cmd.Hostname = userHostMatch[2]
		cmd.Host = cmd.Hostname
		return cmd, true
	}

	// Try hostname only (no user specified)
	if hostMatch := hostOnlyRegex.FindStringSubmatch(argsWithoutFlags); hostMatch != nil {
		hostname := hostMatch[1]
		// Skip common non-hostname arguments
		if hostname == "-v" || hostname == "-vv" || hostname == "-vvv" ||
			strings.HasPrefix(hostname, "-") {
			return Command{}, false
		}
		cmd.Hostname = hostname
		cmd.Host = hostname
		return cmd, true
	}

	return Command{}, false
}

// ConvertToHostConfig converts a Command to a HostConfig.
func (c Command) ConvertToHostConfig() HostConfig {
	config := HostConfig{
		Host:     c.Host,
		Hostname: c.Hostname,
		Port:     c.Port,
		User:     c.User,
	}

	if c.IdentityFile != "" {
		config.IdentityFile = c.IdentityFile
	}

	return config
}
