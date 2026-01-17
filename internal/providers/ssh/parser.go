package ssh

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"sync"

	"github.com/kevinburke/ssh_config"
)

var (
	// mu protects concurrent config modifications.
	mu              sync.Mutex
	errConfigNil    = errors.New("config cannot be nil")
	errHostNotFound = errors.New("host not found")
)

// ParseConfig reads and parses an SSH configuration file.
func ParseConfig(path string) (*ssh_config.Config, error) {
	f, err := os.Open(path) // #nosec G304 - path is user-controlled by design
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			err = closeErr
		}
	}()

	cfg, err := ssh_config.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	return cfg, nil
}

// FindHost searches for a host by hostname pattern in the configuration.
// Returns the host if found, or nil if not found.
func FindHost(cfg *ssh_config.Config, hostname string) *ssh_config.Host {
	if cfg == nil {
		return nil
	}

	for _, host := range cfg.Hosts {
		if host.Matches(hostname) {
			return host
		}
	}

	return nil
}

// FindHostExact searches for a host by exact pattern match (not wildcard).
// Returns the host if found, or nil if not found.
func FindHostExact(cfg *ssh_config.Config, pattern string) *ssh_config.Host {
	if cfg == nil {
		return nil
	}

	for _, host := range cfg.Hosts {
		for _, p := range host.Patterns {
			if p.String() == pattern {
				return host
			}
		}
	}

	return nil
}

// AddHost adds a new host configuration to the config.
// If the host already exists, it returns an error.
func AddHost(cfg *ssh_config.Config, hostConfig HostConfig) error {
	mu.Lock()
	defer mu.Unlock()

	if cfg == nil {
		return errConfigNil
	}

	// Check if host with exact pattern already exists
	existingHost := FindHostExact(cfg, hostConfig.Host)
	if existingHost != nil {
		return fmt.Errorf("host %s already exists", hostConfig.Host)
	}

	// Create new host
	pattern, err := ssh_config.NewPattern(hostConfig.Host)
	if err != nil {
		return fmt.Errorf("failed to create pattern: %w", err)
	}

	newHost := &ssh_config.Host{
		Patterns: []*ssh_config.Pattern{pattern},
		Nodes:    []ssh_config.Node{},
	}

	// Add standard directives
	if hostConfig.Hostname != "" {
		newHost.Nodes = append(newHost.Nodes, &ssh_config.KV{
			Key:   "HostName",
			Value: hostConfig.Hostname,
		})
	}

	if hostConfig.User != "" {
		newHost.Nodes = append(newHost.Nodes, &ssh_config.KV{
			Key:   "User",
			Value: hostConfig.User,
		})
	}

	if hostConfig.Port > 0 {
		newHost.Nodes = append(newHost.Nodes, &ssh_config.KV{
			Key:   "Port",
			Value: strconv.Itoa(hostConfig.Port),
		})
	}

	if hostConfig.IdentityAgent != "" {
		newHost.Nodes = append(newHost.Nodes, &ssh_config.KV{
			Key:   "IdentityAgent",
			Value: hostConfig.IdentityAgent,
		})
	}

	if hostConfig.IdentityFile != "" {
		newHost.Nodes = append(newHost.Nodes, &ssh_config.KV{
			Key:   "IdentityFile",
			Value: hostConfig.IdentityFile,
		})
	}

	// Add additional options
	for _, key := range sortedOptionKeys(hostConfig.Options) {
		newHost.Nodes = append(newHost.Nodes, &ssh_config.KV{
			Key:   key,
			Value: hostConfig.Options[key],
		})
	}

	cfg.Hosts = append(cfg.Hosts, newHost)

	return nil
}

// UpdateHost updates an existing host configuration.
// If the host doesn't exist, it returns an error.
func UpdateHost(cfg *ssh_config.Config, hostConfig HostConfig) error {
	mu.Lock()
	defer mu.Unlock()

	if cfg == nil {
		return errConfigNil
	}

	// Find existing host by exact pattern match
	existingHost := FindHostExact(cfg, hostConfig.Host)
	if existingHost == nil {
		return fmt.Errorf("host %s not found", hostConfig.Host)
	}

	// Clear existing nodes
	existingHost.Nodes = []ssh_config.Node{}

	// Add updated directives
	if hostConfig.Hostname != "" {
		existingHost.Nodes = append(existingHost.Nodes, &ssh_config.KV{
			Key:   "HostName",
			Value: hostConfig.Hostname,
		})
	}

	if hostConfig.User != "" {
		existingHost.Nodes = append(existingHost.Nodes, &ssh_config.KV{
			Key:   "User",
			Value: hostConfig.User,
		})
	}

	if hostConfig.Port > 0 {
		existingHost.Nodes = append(existingHost.Nodes, &ssh_config.KV{
			Key:   "Port",
			Value: strconv.Itoa(hostConfig.Port),
		})
	}

	if hostConfig.IdentityAgent != "" {
		existingHost.Nodes = append(existingHost.Nodes, &ssh_config.KV{
			Key:   "IdentityAgent",
			Value: hostConfig.IdentityAgent,
		})
	}

	if hostConfig.IdentityFile != "" {
		existingHost.Nodes = append(existingHost.Nodes, &ssh_config.KV{
			Key:   "IdentityFile",
			Value: hostConfig.IdentityFile,
		})
	}

	// Add additional options
	for _, key := range sortedOptionKeys(hostConfig.Options) {
		existingHost.Nodes = append(existingHost.Nodes, &ssh_config.KV{
			Key:   key,
			Value: hostConfig.Options[key],
		})
	}

	return nil
}

// RemoveHost removes a host from the configuration.
func RemoveHost(cfg *ssh_config.Config, hostname string) error {
	mu.Lock()
	defer mu.Unlock()

	if cfg == nil {
		return errConfigNil
	}

	// Find and remove host
	for i, host := range cfg.Hosts {
		if host.Matches(hostname) {
			cfg.Hosts = append(cfg.Hosts[:i], cfg.Hosts[i+1:]...)
			return nil
		}
	}

	return errHostNotFound
}

// WriteConfig writes the configuration to a file with proper SSH permissions (0600).
func WriteConfig(cfg *ssh_config.Config, path string) error {
	mu.Lock()
	defer mu.Unlock()

	if cfg == nil {
		return errConfigNil
	}

	// Convert config to string
	content := cfg.String()

	// Write with SSH-required permissions (0600)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// FindHostsByPatterns returns a map of exact host patterns to host blocks.
func FindHostsByPatterns(cfg *ssh_config.Config) map[string]*ssh_config.Host {
	if cfg == nil {
		return map[string]*ssh_config.Host{}
	}

	patternMap := make(map[string]*ssh_config.Host)
	for _, host := range cfg.Hosts {
		for _, pattern := range host.Patterns {
			patternMap[pattern.String()] = host
		}
	}

	return patternMap
}

// UpsertGlobalOptions ensures Host * contains the provided options.
func UpsertGlobalOptions(cfg *ssh_config.Config, options map[string]string) (bool, error) {
	mu.Lock()
	defer mu.Unlock()

	if cfg == nil {
		return false, errConfigNil
	}

	if len(options) == 0 {
		return false, nil
	}

	globalHost := FindHostExact(cfg, "*")
	if globalHost == nil {
		pattern, err := ssh_config.NewPattern("*")
		if err != nil {
			return false, fmt.Errorf("failed to create global pattern: %w", err)
		}
		globalHost = &ssh_config.Host{
			Patterns: []*ssh_config.Pattern{pattern},
			Nodes:    []ssh_config.Node{},
		}
		cfg.Hosts = append(cfg.Hosts, globalHost)
	}

	existing := make(map[string]bool)
	for _, node := range globalHost.Nodes {
		kv, ok := node.(*ssh_config.KV)
		if !ok {
			continue
		}
		existing[kv.Key] = true
	}

	updated := false
	for _, key := range sortedOptionKeys(options) {
		value := options[key]
		if existing[key] {
			updateHostOption(globalHost, key, value)
			updated = true
			continue
		}
		globalHost.Nodes = append(globalHost.Nodes, &ssh_config.KV{
			Key:   key,
			Value: value,
		})
		updated = true
	}

	return updated, nil
}

func updateHostOption(host *ssh_config.Host, key, value string) {
	for _, node := range host.Nodes {
		kv, ok := node.(*ssh_config.KV)
		if !ok {
			continue
		}
		if kv.Key == key {
			kv.Value = value
			return
		}
	}
}

func sortedOptionKeys(options map[string]string) []string {
	keys := make([]string, 0, len(options))
	for key := range options {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// GetHostValue retrieves a single configuration value for a host.
func GetHostValue(cfg *ssh_config.Config, hostname, key string) (string, error) {
	if cfg == nil {
		return "", errConfigNil
	}

	value, err := cfg.Get(hostname, key)
	if err != nil {
		return "", fmt.Errorf("failed to get value: %w", err)
	}

	return value, nil
}

// GetHostValues retrieves all configuration values for a key (useful for IdentityFile).
func GetHostValues(cfg *ssh_config.Config, hostname, key string) ([]string, error) {
	if cfg == nil {
		return nil, errConfigNil
	}

	values, err := cfg.GetAll(hostname, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get values: %w", err)
	}

	return values, nil
}
