package ssh

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
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

	content := renderConfig(cfg)

	// Write with SSH-required permissions (0600)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

func renderConfig(cfg *ssh_config.Config) string {
	var buf strings.Builder

	buf.WriteString("# This file was generated automatically. Do not edit manually.\n")

	includes, topLevel, hosts, wildcardHost := separateConfigSections(cfg)

	if len(includes) > 0 {
		buf.WriteString("\n")
		for _, item := range includes {
			buf.WriteString(item)
			buf.WriteString("\n")
		}
		buf.WriteString("\n")
	}

	if len(topLevel) > 0 {
		buf.WriteString("# Global SSH settings\n")
		for _, item := range topLevel {
			buf.WriteString(item)
			buf.WriteString("\n")
		}
	}

	ordered := orderedHosts(hosts)
	for _, host := range ordered {
		if host == nil {
			continue
		}
		buf.WriteString("\n")
		buf.WriteString(formatHost(host))
	}

	if wildcardHost != nil && len(wildcardHost.Nodes) > 0 {
		buf.WriteString("\n")
		buf.WriteString(formatHost(wildcardHost))
	}

	return buf.String()
}

func separateConfigSections(cfg *ssh_config.Config) ([]string, []string, []*ssh_config.Host, *ssh_config.Host) {
	if cfg == nil {
		return nil, nil, nil, nil
	}

	includes := make([]string, 0)
	topLevel := make([]string, 0)
	hosts := make([]*ssh_config.Host, 0, len(cfg.Hosts))
	var wildcardHost *ssh_config.Host
	seenTopLevel := make(map[string]bool)

	for _, host := range cfg.Hosts {
		if host == nil {
			continue
		}

		if len(host.Patterns) == 0 {
			extractTopLevelNodes(host, &includes, &topLevel)
			continue
		}

		if isGlobalWildcard(host) {
			newWildcard := extractGlobalWildcard(host, &includes, &topLevel, seenTopLevel)
			if newWildcard != nil {
				if wildcardHost == nil {
					wildcardHost = newWildcard
				} else {
					// Merge nodes from multiple wildcard hosts
					wildcardHost.Nodes = append(wildcardHost.Nodes, newWildcard.Nodes...)
				}
			}
			continue
		}

		hosts = append(hosts, host)
	}

	return includes, topLevel, hosts, wildcardHost
}

func extractTopLevelNodes(host *ssh_config.Host, includes *[]string, topLevel *[]string) {
	var pendingComments []string

	for _, node := range host.Nodes {
		// Collect comments
		if empty, ok := node.(*ssh_config.Empty); ok && empty.Comment != "" {
			line := formatNode(node, false)
			if line != "" && !isGeneratedComment(line) {
				pendingComments = append(pendingComments, line)
			}
			continue
		}

		if isIncludeNode(node) {
			// Add pending comments to includes section
			*includes = append(*includes, pendingComments...)
			pendingComments = nil

			line := formatNodeWithComments(node, false)
			if line != "" && !isGeneratedComment(line) {
				*includes = append(*includes, line)
			}
			continue
		}

		// Non-include nodes: add pending comments to topLevel first
		*topLevel = append(*topLevel, pendingComments...)
		pendingComments = nil

		line := formatNode(node, false)
		if line != "" && !isGeneratedComment(line) {
			*topLevel = append(*topLevel, line)
		}
	}

	// Add any remaining comments to topLevel
	*topLevel = append(*topLevel, pendingComments...)
}

func isIncludeNode(node ssh_config.Node) bool {
	switch n := node.(type) {
	case *ssh_config.Include:
		return true
	case *ssh_config.Empty:
		return false
	case *ssh_config.KV:
		return strings.ToLower(n.Key) == "include"
	default:
		return false
	}
}

func formatNodeWithComments(node ssh_config.Node, indent bool) string {
	switch n := node.(type) {
	case *ssh_config.Include:
		if indent {
			return "    " + n.String()
		}
		return n.String()
	default:
		return formatNode(node, indent)
	}
}

func extractGlobalWildcard(host *ssh_config.Host, includes *[]string, topLevel *[]string, seenTopLevel map[string]bool) *ssh_config.Host {
	wildcardNodes := make([]ssh_config.Node, 0)
	var pendingComments []string

	for _, node := range host.Nodes {
		// Collect comments
		if empty, ok := node.(*ssh_config.Empty); ok && empty.Comment != "" {
			line := formatNode(node, false)
			if line != "" && !isGeneratedComment(line) {
				pendingComments = append(pendingComments, line)
			}
			continue
		}

		// Handle Include nodes
		if isIncludeNode(node) {
			// Add pending comments to includes section
			*includes = append(*includes, pendingComments...)
			pendingComments = nil

			line := formatNodeWithComments(node, false)
			if line != "" && !isGeneratedComment(line) {
				*includes = append(*includes, line)
			}
			continue
		}

		// Check if this is a host-specific setting that should stay in Host *
		if isHostSpecificSetting(node) {
			// Add pending comments as formatted strings (they're already processed)
			// Note: comments before host-specific settings are discarded for now
			pendingComments = nil
			wildcardNodes = append(wildcardNodes, node)
			continue
		}

		// Regular global settings
		*topLevel = append(*topLevel, pendingComments...)
		pendingComments = nil

		line := formatNode(node, false)
		if line != "" && !isGeneratedComment(line) && !seenTopLevel[line] {
			*topLevel = append(*topLevel, line)
			seenTopLevel[line] = true
		}
	}

	// Add any remaining comments to topLevel
	*topLevel = append(*topLevel, pendingComments...)

	if len(wildcardNodes) > 0 {
		pattern, _ := ssh_config.NewPattern("*")
		return &ssh_config.Host{
			Patterns: []*ssh_config.Pattern{pattern},
			Nodes:    wildcardNodes,
		}
	}

	return nil
}

func isHostSpecificSetting(node ssh_config.Node) bool {
	kv, ok := node.(*ssh_config.KV)
	if !ok {
		return false
	}

	hostSpecificKeys := []string{
		"IdentityAgent",
		"ProxyCommand",
		"ProxyJump",
		"LocalForward",
		"RemoteForward",
		"DynamicForward",
	}

	key := strings.ToLower(kv.Key)
	for _, specificKey := range hostSpecificKeys {
		if key == strings.ToLower(specificKey) {
			return true
		}
	}

	return false
}

func isGlobalWildcard(host *ssh_config.Host) bool {
	if host == nil || len(host.Patterns) == 0 {
		return false
	}

	for _, pattern := range host.Patterns {
		if pattern != nil && pattern.String() == "*" {
			return true
		}
	}

	return false
}

func isGeneratedComment(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.Contains(trimmed, "This file was generated automatically") ||
		strings.Contains(trimmed, "Top level SSH settings") ||
		strings.Contains(trimmed, "Global SSH settings") ||
		strings.Contains(trimmed, "Global settings")
}

func formatHost(host *ssh_config.Host) string {
	var buf strings.Builder

	buf.WriteString("Host ")
	for i, pattern := range host.Patterns {
		if i > 0 {
			buf.WriteString(" ")
		}
		buf.WriteString(pattern.String())
	}
	buf.WriteString("\n")

	for _, node := range host.Nodes {
		line := formatNode(node, true)
		if line != "" {
			buf.WriteString(line)
			buf.WriteString("\n")
		}
	}

	return buf.String()
}

func formatNode(node ssh_config.Node, indent bool) string {
	switch n := node.(type) {
	case *ssh_config.KV:
		if indent {
			return fmt.Sprintf("    %s %s", n.Key, n.Value)
		}
		return fmt.Sprintf("%s %s", n.Key, n.Value)
	case *ssh_config.Empty:
		if n.Comment != "" {
			if indent {
				return "    #" + n.Comment
			}
			return "#" + n.Comment
		}
		return ""
	case *ssh_config.Include:
		if indent {
			return "    " + n.String()
		}
		return n.String()
	default:
		return ""
	}
}

func orderedHosts(hosts []*ssh_config.Host) []*ssh_config.Host {
	if len(hosts) == 0 {
		return hosts
	}

	nonWildcard := make([]*ssh_config.Host, 0, len(hosts))
	wildcard := make([]*ssh_config.Host, 0, len(hosts))
	for _, host := range hosts {
		if host == nil {
			continue
		}
		if hasWildcardPattern(host) {
			wildcard = append(wildcard, host)
			continue
		}
		nonWildcard = append(nonWildcard, host)
	}

	sort.SliceStable(nonWildcard, func(i, j int) bool {
		return hostSortKey(nonWildcard[i]) < hostSortKey(nonWildcard[j])
	})

	sort.SliceStable(wildcard, func(i, j int) bool {
		return hostSortKey(wildcard[i]) < hostSortKey(wildcard[j])
	})

	return append(nonWildcard, wildcard...)
}

func hostSortKey(host *ssh_config.Host) string {
	if host == nil || len(host.Patterns) == 0 || host.Patterns[0] == nil {
		return ""
	}

	return strings.ToLower(host.Patterns[0].String())
}

func hasWildcardPattern(host *ssh_config.Host) bool {
	if host == nil {
		return false
	}

	for _, pattern := range host.Patterns {
		if pattern == nil {
			continue
		}
		if strings.ContainsAny(pattern.String(), "*?") {
			return true
		}
	}

	return false
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

	globalHosts := findHostsByPattern(cfg, "*")
	if len(globalHosts) == 0 {
		pattern, err := ssh_config.NewPattern("*")
		if err != nil {
			return false, fmt.Errorf("failed to create global pattern: %w", err)
		}
		globalHosts = []*ssh_config.Host{
			{
				Patterns: []*ssh_config.Pattern{pattern},
				Nodes:    []ssh_config.Node{},
			},
		}
		cfg.Hosts = append(cfg.Hosts, globalHosts[0])
	}

	updated := false
	for _, globalHost := range globalHosts {
		if applyOptions(globalHost, options) {
			updated = true
		}
	}

	return updated, nil
}

func findHostsByPattern(cfg *ssh_config.Config, pattern string) []*ssh_config.Host {
	if cfg == nil {
		return nil
	}

	matches := make([]*ssh_config.Host, 0)
	for _, host := range cfg.Hosts {
		if host == nil {
			continue
		}
		for _, hostPattern := range host.Patterns {
			if hostPattern == nil {
				continue
			}
			if hostPattern.String() == pattern {
				matches = append(matches, host)
				break
			}
		}
	}

	return matches
}

func applyOptions(host *ssh_config.Host, options map[string]string) bool {
	if host == nil {
		return false
	}

	existing := make(map[string]bool)
	for _, node := range host.Nodes {
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
			updateHostOption(host, key, value)
			updated = true
			continue
		}
		host.Nodes = append(host.Nodes, &ssh_config.KV{
			Key:   key,
			Value: value,
		})
		updated = true
	}

	return updated
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
