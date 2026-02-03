package aws

import (
	"errors"
	"fmt"
	"html/template"
	"sort"
	"strings"
)

const (
	profileSectionPrefix = "profile "
	ssoSessionSection    = "sso-session"
)

var errProfileTemplateEmpty = errors.New("profile template cannot be empty")

type generatedProfile struct {
	AccountID string
	Name      string
	RoleName  string
}

// BuildConfigContent renders the AWS shared config content for discovered profiles.
func BuildConfigContent(cfg *Config, profiles []DiscoveredProfile) (string, []string, error) {
	content, _, warnings, err := BuildGeneratedConfigContent(cfg, profiles)
	if err != nil {
		return "", nil, err
	}

	return content, warnings, nil
}

// BuildGeneratedConfigContent renders config content and returns generated profile names.
func BuildGeneratedConfigContent(cfg *Config, profiles []DiscoveredProfile) (string, []string, []string, error) {
	if cfg == nil {
		return "", nil, nil, errors.New("aws config is nil")
	}

	if strings.TrimSpace(cfg.SSO.SessionName) == "" {
		return "", nil, nil, errors.New("sso session name is empty")
	}

	profileNames, lookup, err := buildProfileIndex(cfg, profiles)
	if err != nil {
		return "", nil, nil, err
	}

	sourceProfiles := make(map[string]bool, len(lookup))
	for name := range lookup {
		sourceProfiles[name] = true
	}

	roleChainNames, roleChainLookup, warnings, err := buildRoleChainIndex(cfg, sourceProfiles)
	if err != nil {
		return "", nil, nil, err
	}

	generatedNames := make([]string, 0, len(profileNames)+len(roleChainNames))
	generatedNames = append(generatedNames, profileNames...)
	generatedNames = append(generatedNames, roleChainNames...)

	builder := &strings.Builder{}
	if err := writeSSOSession(builder, cfg); err != nil {
		return "", nil, nil, err
	}

	for _, name := range profileNames {
		profile := lookup[name]
		writeProfileSection(builder, cfg, profile)
	}

	for _, name := range roleChainNames {
		profile := roleChainLookup[name]
		writeRoleChainSection(builder, cfg, profile)
	}

	return strings.TrimRight(builder.String(), "\n"), generatedNames, warnings, nil
}

func buildProfileIndex(cfg *Config, profiles []DiscoveredProfile) ([]string, map[string]generatedProfile, error) {
	if cfg == nil {
		return nil, nil, errors.New("aws config is nil")
	}

	if strings.TrimSpace(cfg.ProfileTemplate) == "" {
		return nil, nil, errProfileTemplateEmpty
	}

	profileTemplate, err := template.New("profile").Option("missingkey=error").Parse(cfg.ProfileTemplate)
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
	var builder strings.Builder
	if err := tmpl.Execute(&builder, newTemplateData(profile)); err != nil {
		return "", fmt.Errorf("execute profile template: %w", err)
	}

	return builder.String(), nil
}

func newTemplateData(profile DiscoveredProfile) map[string]string {
	return map[string]string{
		"AccountID":    profile.AccountID,
		"AccountName":  profile.AccountName,
		"RoleName":     profile.RoleName,
		"SSORegion":    profile.SSORegion,
		"account":      profile.AccountName,
		"account_id":   profile.AccountID,
		"account_name": profile.AccountName,
		"role":         profile.RoleName,
		"role_name":    profile.RoleName,
		"sso_region":   profile.SSORegion,
	}
}

type roleChainProfile struct {
	Name          string
	Region        string
	RoleARN       string
	SourceProfile string
}

func buildRoleChainIndex(cfg *Config, sourceProfiles map[string]bool) ([]string, map[string]roleChainProfile, []string, error) {
	if cfg == nil {
		return nil, nil, nil, errors.New("aws config is nil")
	}

	profileMap := make(map[string]roleChainProfile)
	order := make([]string, 0, len(cfg.RoleChains))
	warnings := []string{}
	for i, chain := range cfg.RoleChains {
		name := strings.TrimSpace(chain.Name)
		if name == "" {
			return nil, nil, nil, fmt.Errorf("role chain name is empty at index %d", i)
		}
		name = cfg.ProfilePrefix + name

		sourceProfile := strings.TrimSpace(chain.SourceProfile)
		if sourceProfile == "" {
			return nil, nil, nil, fmt.Errorf("role chain source_profile is empty for %s", name)
		}

		roleARN := strings.TrimSpace(chain.RoleARN)
		if roleARN == "" {
			return nil, nil, nil, fmt.Errorf("role chain role_arn is empty for %s", name)
		}

		if _, ok := sourceProfiles[sourceProfile]; !ok {
			warnings = append(warnings, fmt.Sprintf("source_profile %q not found in discovered profiles", sourceProfile))
		}

		if _, exists := profileMap[name]; !exists {
			order = append(order, name)
		}
		profileMap[name] = roleChainProfile{
			Name:          name,
			Region:        strings.TrimSpace(chain.Region),
			RoleARN:       roleARN,
			SourceProfile: sourceProfile,
		}
	}

	sort.Strings(order)
	return order, profileMap, warnings, nil
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
	writeProfileEntry(builder, cfg, profile.Name, profile.AccountID, profile.RoleName)
	writeMarker(builder, cfg)
	builder.WriteString("\n")
}

func writeRoleChainSection(builder *strings.Builder, cfg *Config, profile roleChainProfile) {
	writeSectionHeader(builder, profileSectionPrefix+profile.Name)
	writeKeyValue(builder, "source_profile", profile.SourceProfile)
	writeKeyValue(builder, "role_arn", profile.RoleARN)
	if profile.Region != "" {
		writeKeyValue(builder, "region", profile.Region)
	}
	writeMarker(builder, cfg)
	builder.WriteString("\n")
}

// BuildCredentialProcessContent renders credentials file content using credential_process.
func BuildCredentialProcessContent(cfg *Config, profiles []DiscoveredProfile) (string, []string, []string, error) {
	if cfg == nil {
		return "", nil, nil, errors.New("aws config is nil")
	}

	profileNames, lookup, err := buildProfileIndex(cfg, profiles)
	if err != nil {
		return "", nil, nil, err
	}

	sourceProfiles := make(map[string]bool, len(lookup))
	for name := range lookup {
		sourceProfiles[name] = true
	}

	roleChainNames, roleChainLookup, warnings, err := buildRoleChainIndex(cfg, sourceProfiles)
	if err != nil {
		return "", nil, nil, err
	}

	builder := &strings.Builder{}
	generatedNames := make([]string, 0, len(profileNames)+len(roleChainNames))
	for _, name := range profileNames {
		profile := lookup[name]
		writeCredentialProcessSection(builder, profile.Name)
		generatedNames = append(generatedNames, profile.Name)
	}

	for _, name := range roleChainNames {
		profile := roleChainLookup[name]
		writeCredentialProcessSection(builder, profile.Name)
		generatedNames = append(generatedNames, profile.Name)
	}

	return strings.TrimRight(builder.String(), "\n"), generatedNames, warnings, nil
}

func writeCredentialProcessSection(builder *strings.Builder, profileName string) {
	writeSectionHeader(builder, profileName)
	writeKeyValue(builder, "credential_process", "granted credential-process --profile "+profileName)
	builder.WriteString("\n")
}

func writeProfileEntry(builder *strings.Builder, cfg *Config, profileName, accountID, roleName string) {
	if cfg.UseCredentialProcess {
		writeKeyValue(builder, "credential_process", "granted credential-process --profile "+profileName)
		return
	}
	writeKeyValue(builder, "sso_session", cfg.SSO.SessionName)
	writeKeyValue(builder, "sso_account_id", accountID)
	writeKeyValue(builder, "sso_role_name", roleName)
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

func writeMarker(builder *strings.Builder, cfg *Config) {
	if cfg == nil {
		return
	}
	marker := strings.TrimSpace(cfg.MarkerKey)
	if marker == "" {
		return
	}
	writeKeyValue(builder, marker, "true")
}
