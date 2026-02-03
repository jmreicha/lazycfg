package aws

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

type configSection struct {
	Header string
	Keys   []string
	Name   string
	Raw    string
}

func mergeConfigContent(path, generated string, generatedNames []string, markerKey, sessionName string) (string, error) {
	markerKey = strings.TrimSpace(markerKey)
	if markerKey == "" {
		return generated, nil
	}

	existing, err := readConfigFile(path)
	if err != nil {
		return "", err
	}

	if strings.TrimSpace(existing) == "" {
		return generated, nil
	}

	existingSections := parseConfigSections(existing)
	generatedSections := parseConfigSections(generated)

	generatedSet := make(map[string]bool, len(generatedNames))
	for _, name := range generatedNames {
		if strings.TrimSpace(name) == "" {
			continue
		}
		generatedSet[name] = true
	}

	keptExisting := make([]configSection, 0, len(existingSections))
	for _, section := range existingSections {
		if isGeneratedSession(section, sessionName) {
			continue
		}
		if section.Name == "" {
			keptExisting = append(keptExisting, section)
			continue
		}
		if generatedSet[section.Name] {
			continue
		}
		if section.HasKey(markerKey) {
			continue
		}
		keptExisting = append(keptExisting, section)
	}

	keptExisting = append(keptExisting, generatedSections...)
	return joinSections(keptExisting), nil
}

func readConfigFile(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errors.New("config path is empty")
	}

	// #nosec G304 -- config path is user-configurable
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", fmt.Errorf("read config file: %w", err)
	}

	return string(data), nil
}

func parseConfigSections(content string) []configSection {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(content, "\n")
	sections := []configSection{}
	current := configSection{}
	flush := func() {
		if current.Raw == "" && current.Header == "" && len(current.Keys) == 0 {
			return
		}
		sections = append(sections, current)
		current = configSection{}
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			flush()
			current.Header = strings.TrimSpace(trimmed[1 : len(trimmed)-1])
			current.Name = normalizeProfileName(current.Header)
			current.Raw = line + "\n"
			continue
		}

		if trimmed != "" && strings.Contains(trimmed, "=") && !strings.HasPrefix(trimmed, ";") && !strings.HasPrefix(trimmed, "#") {
			parts := strings.SplitN(trimmed, "=", 2)
			key := strings.TrimSpace(parts[0])
			if key != "" {
				current.Keys = append(current.Keys, key)
			}
		}
		current.Raw += line + "\n"
	}
	flush()

	return sections
}

func normalizeProfileName(header string) string {
	name := strings.TrimSpace(header)
	prefix := "profile "
	if strings.HasPrefix(strings.ToLower(name), prefix) {
		name = strings.TrimSpace(name[len(prefix):])
		return name
	}
	return ""
}

func (section configSection) HasKey(key string) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}
	for _, existing := range section.Keys {
		if strings.EqualFold(existing, key) {
			return true
		}
	}
	return false
}

func joinSections(sections []configSection) string {
	if len(sections) == 0 {
		return ""
	}
	builder := strings.Builder{}
	for _, section := range sections {
		builder.WriteString(section.Raw)
	}
	content := strings.TrimRight(builder.String(), "\n")
	if content == "" {
		return ""
	}
	return content
}

func isGeneratedSession(section configSection, sessionName string) bool {
	sessionName = strings.TrimSpace(sessionName)
	if sessionName == "" {
		return false
	}
	header := strings.TrimSpace(section.Header)
	if header == "" {
		return false
	}
	return strings.EqualFold(header, fmt.Sprintf("%s %s", "sso-session", sessionName))
}
