package cli

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"testing"

	"github.com/jmreicha/cfgctl/internal/core"
	"github.com/jmreicha/cfgctl/internal/providers/aws"
	"github.com/jmreicha/cfgctl/internal/providers/kubernetes"
)

type commandProvider struct {
	name         string
	cleanErr     error
	validateErr  error
	cleanCalled  bool
	validChecked bool
}

func (p *commandProvider) Name() string {
	return p.name
}

func (p *commandProvider) Validate(_ context.Context) error {
	p.validChecked = true
	return p.validateErr
}

func (p *commandProvider) Generate(_ context.Context, _ *core.GenerateOptions) (*core.Result, error) {
	return &core.Result{Provider: p.name}, nil
}

func (p *commandProvider) Backup(_ context.Context) (string, error) {
	return "", nil
}

func (p *commandProvider) Restore(_ context.Context, _ string) error {
	return nil
}

func (p *commandProvider) Clean(_ context.Context) error {
	p.cleanCalled = true
	return p.cleanErr
}

func setupCommandEngine(t *testing.T, providers ...core.Provider) {
	t.Helper()

	logger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
	registry = core.NewRegistry()
	backupManager = core.NewBackupManager("")
	config = core.NewConfig()
	engine = core.NewEngine(registry, backupManager, config, logger)

	for _, provider := range providers {
		if err := registry.Register(provider); err != nil {
			t.Fatalf("failed to register provider: %v", err)
		}
	}
}

func TestInitializeComponents(t *testing.T) {
	cfgFile = ""
	dryRun = false
	debug = false
	noBackup = false
	sshConfigPath = ""
	verbose = false

	if err := initializeComponents(); err != nil {
		t.Fatalf("initializeComponents failed: %v", err)
	}

	if registry == nil {
		t.Fatal("expected registry to be initialized")
	}

	providers := registry.List()
	if len(providers) != 4 {
		t.Fatalf("expected four providers, got %v", providers)
	}
	if providers[0] != "aws" || providers[1] != "granted" || providers[2] != "kubernetes" || providers[3] != "ssh" {
		t.Fatalf("expected aws, granted, kubernetes, and ssh providers, got %v", providers)
	}
}

func TestNewRootCmdFlags(t *testing.T) {
	cmd := NewRootCmd("1.0.0")
	flags := cmd.PersistentFlags()

	for _, name := range []string{"config", "debug", "dry-run", "no-backup", "ssh-config-path", "verbose"} {
		if flags.Lookup(name) == nil {
			t.Fatalf("expected %s flag", name)
		}
	}
}

func TestApplyKubernetesCLIOverrides(t *testing.T) {
	prevKubeDemo := kubeDemo
	prevKubeMerge := kubeMerge
	prevKubeMergeOnly := kubeMergeOnly
	prevKubeProfiles := kubeProfiles
	prevKubeRegions := kubeRegions
	defer func() {
		kubeDemo = prevKubeDemo
		kubeMerge = prevKubeMerge
		kubeMergeOnly = prevKubeMergeOnly
		kubeProfiles = prevKubeProfiles
		kubeRegions = prevKubeRegions
	}()

	kubeDemo = true
	kubeMerge = true
	kubeMergeOnly = false
	kubeProfiles = "prod,dev"
	kubeRegions = "us-west-2,us-east-1"

	cfg := kubernetes.DefaultConfig()
	applyKubernetesCLIOverrides(cfg)

	if !cfg.Demo {
		t.Fatal("expected demo to be enabled")
	}
	if !cfg.MergeEnabled {
		t.Fatal("expected merge to be enabled")
	}
	if cfg.MergeOnly {
		t.Fatal("expected merge-only to be false")
	}
	if !reflect.DeepEqual(cfg.AWS.Profiles, []string{"dev", "prod"}) {
		t.Fatalf("profiles = %v", cfg.AWS.Profiles)
	}
	if !reflect.DeepEqual(cfg.AWS.Regions, []string{"us-east-1", "us-west-2"}) {
		t.Fatalf("regions = %v", cfg.AWS.Regions)
	}
}

func TestApplyKubernetesCLIOverridesMergeOnly(t *testing.T) {
	prevKubeMerge := kubeMerge
	prevKubeMergeOnly := kubeMergeOnly
	defer func() {
		kubeMerge = prevKubeMerge
		kubeMergeOnly = prevKubeMergeOnly
	}()

	kubeMerge = false
	kubeMergeOnly = true

	cfg := kubernetes.DefaultConfig()
	applyKubernetesCLIOverrides(cfg)

	if !cfg.MergeOnly {
		t.Fatal("expected merge-only to be true")
	}
	if !cfg.MergeEnabled {
		t.Fatal("expected merge to be enabled")
	}
}

func TestInitializeComponentsAWSCLIOverrides(t *testing.T) {
	cfgFile = ""
	dryRun = false
	noBackup = false
	sshConfigPath = ""
	verbose = false
	awsCredentialProcess = true
	awsCredentials = true
	awsDemo = true
	awsPrefix = "team-"
	awsPrune = true
	awsRoleFilters = "AdminAccess,ReadOnly"
	awsTemplate = "{{ .account }}-{{ .role }}"

	if err := initializeComponents(); err != nil {
		t.Fatalf("initializeComponents failed: %v", err)
	}

	providerConfig := config.GetProviderConfig(aws.ProviderName)
	awsConfig, ok := providerConfig.(*aws.Config)
	if !ok {
		t.Fatalf("expected aws config, got %T", providerConfig)
	}

	if !awsConfig.UseCredentialProcess {
		t.Fatal("expected credential_process enabled")
	}
	if !awsConfig.GenerateCredentials {
		t.Fatal("expected credentials generation enabled")
	}
	if !awsConfig.Demo {
		t.Fatal("expected demo enabled")
	}
	if awsConfig.ProfilePrefix != "team-" {
		t.Fatalf("profile prefix = %q", awsConfig.ProfilePrefix)
	}
	if !awsConfig.Prune {
		t.Fatal("expected prune enabled")
	}
	if awsConfig.ProfileTemplate != "{{ .account }}-{{ .role }}" {
		t.Fatalf("profile template = %q", awsConfig.ProfileTemplate)
	}
	if !reflect.DeepEqual(awsConfig.Roles, []string{"AdminAccess", "ReadOnly"}) {
		t.Fatalf("roles = %#v", awsConfig.Roles)
	}
}

func TestNewVersionCmd(t *testing.T) {
	cmd := newVersionCmd("1.0.0")
	if err := cmd.Execute(); err != nil {
		t.Fatalf("version command failed: %v", err)
	}
}

func TestListCmd_Empty(t *testing.T) {
	setupCommandEngine(t)

	cmd := newListCmd()
	if err := cmd.Execute(); err != nil {
		t.Fatalf("list command failed: %v", err)
	}
}

func TestListCmd_WithProviders(t *testing.T) {
	setupCommandEngine(t, &commandProvider{name: "alpha"})

	cmd := newListCmd()
	if err := cmd.Execute(); err != nil {
		t.Fatalf("list command failed: %v", err)
	}
}

func TestValidateCmd(t *testing.T) {
	provider := &commandProvider{name: "alpha"}
	setupCommandEngine(t, provider)

	cmd := newValidateCmd()
	if err := cmd.Execute(); err != nil {
		t.Fatalf("validate command failed: %v", err)
	}
	if !provider.validChecked {
		t.Fatal("expected provider validation")
	}
}

func TestValidateCmd_Error(t *testing.T) {
	provider := &commandProvider{name: "alpha", validateErr: errors.New("no")}
	setupCommandEngine(t, provider)

	cmd := newValidateCmd()
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCleanCmd_NoArgs(t *testing.T) {
	setupCommandEngine(t)

	cmd := newCleanCmd()
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCleanCmd_WithArgs(t *testing.T) {
	provider := &commandProvider{name: "alpha"}
	setupCommandEngine(t, provider)

	cmd := newCleanCmd()
	cmd.SetArgs([]string{"alpha"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("clean command failed: %v", err)
	}
	if !provider.cleanCalled {
		t.Fatal("expected clean to be called")
	}
}
