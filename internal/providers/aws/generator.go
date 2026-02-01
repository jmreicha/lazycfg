package aws

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"
	"text/template"
)

const (
	profileSectionPrefix = "profile "
	ssoSessionSection    = "sso-session"
)

type generatedProfile struct {
	AccountID string
	Name      string
	RoleName  string
}

// BuildConfigContent renders the AWS shared config content for discovered profiles.
func BuildConfigContent(cfg *Config, profiles []DiscoveredProfile) (string, error) {
	if cfg == nil {
		return "", errors.New("aws config is nil")
	}

	if strings.TrimSpace(cfg.SSO.SessionName) == "" {
		return "", errors.New("sso session name is empty")
	}

	profileNames, lookup, err := buildProfileIndex(cfg, profiles)
	if err != nil {
		return "", err
	}

	builder := &strings.Builder{}
	if err := writeSSOSession(builder, cfg); err != nil {
		return "", err
	}

	for _, name := range profileNames {
		profile := lookup[name]
		writeProfileSection(builder, cfg, profile)
	}

	return strings.TrimRight(builder.String(), "\n"), nil
}

func buildProfileIndex(cfg *Config, profiles []DiscoveredProfile) ([]string, map[string]generatedProfile, error) {
	if cfg == nil {
		return nil, nil, errors.New("aws config is nil")
	}

	profileTemplate, err := template.New("profile").Parse(cfg.ProfileTemplate)
	if err != nil {
		return nil, nil, fmt.Errorf("parse profile template: %w", err)
	}

	profileMap := make(map[string]generatedProfile)
	order := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		name, err := executeTemplate(profileTemplate, profile)
		if err != nil {
			return nil, nil, err
		}
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, nil, errors.New("generated profile name is empty")
		}
		name = cfg.ProfilePrefix + name
		if _, exists := profileMap[name]; !exists {
			order = append(order, name)
		}
		profileMap[name] = generatedProfile{
			AccountID: profile.AccountID,
			Name:      name,
			RoleName:  profile.RoleName,
		}
	}

	sort.Strings(order)
	return order, profileMap, nil
}

func executeTemplate(tmpl *template.Template, profile DiscoveredProfile) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, profile); err != nil {
		return "", fmt.Errorf("execute profile template: %w", err)
	}

	return buf.String(), nil
}

func writeSSOSession(builder *strings.Builder, cfg *Config) error {
	if cfg == nil {
		return errors.New("aws config is nil")
	}
	if builder == nil {
		return errors.New("config builder is nil")
	}

	writeSectionHeader(builder, fmt.Sprintf("%s %s", ssoSessionSection, cfg.SSO.SessionName))
	writeKeyValue(builder, "sso_start_url", cfg.SSO.StartURL)
	writeKeyValue(builder, "sso_region", cfg.SSO.Region)
	writeKeyValue(builder, "sso_registration_scopes", cfg.SSO.RegistrationScopes)
	builder.WriteString("\n")

	return nil
}

func writeProfileSection(builder *strings.Builder, cfg *Config, profile generatedProfile) {
	writeSectionHeader(builder, profileSectionPrefix+profile.Name)
	writeKeyValue(builder, "sso_session", cfg.SSO.SessionName)
	writeKeyValue(builder, "sso_account_id", profile.AccountID)
	writeKeyValue(builder, "sso_role_name", profile.RoleName)
	builder.WriteString("\n")
}

func writeSectionHeader(builder *strings.Builder, name string) {
	builder.WriteString("[")
	builder.WriteString(name)
	builder.WriteString("]\n")
}

func writeKeyValue(builder *strings.Builder, key, value string) {
	builder.WriteString(key)
	builder.WriteString(" = ")
	builder.WriteString(value)
	builder.WriteString("\n")
}
