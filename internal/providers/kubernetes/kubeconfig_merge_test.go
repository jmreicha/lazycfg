package kubernetes

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd/api"
)

func TestMergeKubeconfigs(t *testing.T) {
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "config")

	existingData, err := readFixture("kubeconfig_existing.yaml")
	if err != nil {
		t.Fatalf("readFixture failed: %v", err)
	}
	if err := writeFixture(outputPath, string(existingData)); err != nil {
		t.Fatalf("writeFixture failed: %v", err)
	}

	mergeDir := filepath.Join(dir, "merge")
	if err := ensureDir(mergeDir); err != nil {
		t.Fatalf("ensureDir failed: %v", err)
	}

	mergeConfigPath := filepath.Join(mergeDir, "extra.yaml")
	extraData, err := readFixture("kubeconfig_extra.yaml")
	if err != nil {
		t.Fatalf("readFixture failed: %v", err)
	}
	if err := writeFixture(mergeConfigPath, string(extraData)); err != nil {
		t.Fatalf("writeFixture failed: %v", err)
	}

	discovered := newKubeconfig()
	discovered.Clusters["existing"] = &api.Cluster{Server: "https://override"}
	discovered.Contexts["existing"] = &api.Context{Cluster: "existing", AuthInfo: "existing"}
	discovered.AuthInfos["existing"] = &api.AuthInfo{}

	mergeConfig := MergeConfig{
		SourceDir:       mergeDir,
		IncludePatterns: []string{"*.yaml"},
	}

	merged, files, err := MergeKubeconfigs(outputPath, mergeConfig, discovered)
	if err != nil {
		t.Fatalf("MergeKubeconfigs failed: %v", err)
	}

	if !reflect.DeepEqual(files, []string{mergeConfigPath}) {
		t.Fatalf("merge files = %v", files)
	}

	if merged.Clusters["existing"].Server != "https://override" {
		t.Errorf("existing server = %q", merged.Clusters["existing"].Server)
	}

	if merged.Clusters["extra"] == nil {
		t.Error("expected extra cluster from merge file")
	}
}

func TestMergeKubeconfigsMissingSource(t *testing.T) {
	dir := t.TempDir()
	mergeConfig := MergeConfig{
		SourceDir:       filepath.Join(dir, "missing"),
		IncludePatterns: []string{"*.yaml"},
	}

	merged, files, err := MergeKubeconfigs(filepath.Join(dir, "output"), mergeConfig, nil)
	if err != nil {
		t.Fatalf("MergeKubeconfigs failed: %v", err)
	}

	if len(files) != 0 {
		t.Fatalf("merge files = %v", files)
	}

	if merged == nil {
		t.Fatal("expected merged config")
	}
}

func TestResolveMergeFiles(t *testing.T) {
	dir := t.TempDir()
	if err := ensureDir(filepath.Join(dir, "nested")); err != nil {
		t.Fatalf("ensureDir failed: %v", err)
	}

	includePath := filepath.Join(dir, "config")
	excludedPath := filepath.Join(dir, "config.bak")
	nestedPath := filepath.Join(dir, "nested", "extra.yaml")

	if err := writeFixture(includePath, ""); err != nil {
		t.Fatalf("writeFixture failed: %v", err)
	}
	if err := writeFixture(excludedPath, ""); err != nil {
		t.Fatalf("writeFixture failed: %v", err)
	}
	if err := writeFixture(nestedPath, ""); err != nil {
		t.Fatalf("writeFixture failed: %v", err)
	}

	mergeConfig := MergeConfig{
		SourceDir:       dir,
		IncludePatterns: []string{"*.yaml", "config"},
		ExcludePatterns: []string{"*.bak"},
	}

	files, err := resolveMergeFiles(mergeConfig)
	if err != nil {
		t.Fatalf("resolveMergeFiles failed: %v", err)
	}

	expected := []string{includePath, nestedPath}
	if !reflect.DeepEqual(files, expected) {
		t.Fatalf("files = %v", files)
	}
}

func TestEnsureKubeconfigDefaults(t *testing.T) {
	t.Run("nil config does not panic", func(_ *testing.T) {
		ensureKubeconfigDefaults(nil)
	})

	t.Run("fills in missing defaults", func(t *testing.T) {
		config := &api.Config{}
		ensureKubeconfigDefaults(config)

		if config.Kind != kubeconfigKind {
			t.Errorf("Kind = %q, want %q", config.Kind, kubeconfigKind)
		}
		if config.APIVersion != "v1" {
			t.Errorf("APIVersion = %q, want %q", config.APIVersion, "v1")
		}
		if config.Clusters == nil {
			t.Error("expected Clusters to be initialized")
		}
		if config.AuthInfos == nil {
			t.Error("expected AuthInfos to be initialized")
		}
		if config.Contexts == nil {
			t.Error("expected Contexts to be initialized")
		}
		if config.Extensions == nil {
			t.Error("expected Extensions to be initialized")
		}
	})

	t.Run("preserves existing values", func(t *testing.T) {
		config := &api.Config{
			Kind:       "CustomKind",
			APIVersion: "v2",
			Clusters:   map[string]*api.Cluster{"existing": {}},
		}
		ensureKubeconfigDefaults(config)

		if config.Kind != "CustomKind" {
			t.Errorf("Kind = %q, want %q", config.Kind, "CustomKind")
		}
		if config.APIVersion != "v2" {
			t.Errorf("APIVersion = %q, want %q", config.APIVersion, "v2")
		}
		if _, ok := config.Clusters["existing"]; !ok {
			t.Error("expected existing cluster to be preserved")
		}
	})
}

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		target  string
		want    bool
	}{
		{
			name:    "empty pattern returns false",
			pattern: "",
			target:  "file.yaml",
			want:    false,
		},
		{
			name:    "empty target returns false",
			pattern: "*.yaml",
			target:  "",
			want:    false,
		},
		{
			name:    "matching pattern",
			pattern: "*.yaml",
			target:  "config.yaml",
			want:    true,
		},
		{
			name:    "non-matching pattern",
			pattern: "*.yaml",
			target:  "config.json",
			want:    false,
		},
		{
			name:    "exact match",
			pattern: "config",
			target:  "config",
			want:    true,
		},
		{
			name:    "invalid pattern returns false",
			pattern: "[",
			target:  "file",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchPattern(tt.pattern, tt.target); got != tt.want {
				t.Errorf("matchPattern(%q, %q) = %v, want %v", tt.pattern, tt.target, got, tt.want)
			}
		})
	}
}

func TestPreferencesEmpty(t *testing.T) {
	tests := []struct {
		name  string
		prefs api.Preferences
		want  bool
	}{
		{
			name:  "empty preferences",
			prefs: api.Preferences{},
			want:  true,
		},
		{
			name:  "colors enabled",
			prefs: api.Preferences{Colors: true},
			want:  false,
		},
		{
			name: "extensions present",
			prefs: api.Preferences{
				Extensions: map[string]runtime.Object{"ext": nil},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := preferencesEmpty(tt.prefs); got != tt.want {
				t.Errorf("preferencesEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMergeKubeconfigsEmptyOutputPath(t *testing.T) {
	dir := t.TempDir()
	mergeDir := filepath.Join(dir, "merge")
	if err := ensureDir(mergeDir); err != nil {
		t.Fatalf("ensureDir failed: %v", err)
	}

	mergeConfig := MergeConfig{
		SourceDir:       mergeDir,
		IncludePatterns: []string{"*.yaml"},
	}

	merged, _, err := MergeKubeconfigs("", mergeConfig, nil)
	if err != nil {
		t.Fatalf("MergeKubeconfigs failed: %v", err)
	}
	if merged == nil {
		t.Fatal("expected non-nil merged config")
	}
}

func TestMergeKubeconfigsWithDiscoveredOnly(t *testing.T) {
	dir := t.TempDir()
	mergeDir := filepath.Join(dir, "merge")
	if err := ensureDir(mergeDir); err != nil {
		t.Fatalf("ensureDir failed: %v", err)
	}

	discovered := newKubeconfig()
	discovered.Clusters["test-cluster"] = &api.Cluster{Server: "https://test.example.com"}
	discovered.Contexts["test-context"] = &api.Context{Cluster: "test-cluster", AuthInfo: "test-user"}
	discovered.AuthInfos["test-user"] = &api.AuthInfo{}

	mergeConfig := MergeConfig{
		SourceDir:       mergeDir,
		IncludePatterns: []string{"*.yaml"},
	}

	merged, _, err := MergeKubeconfigs("", mergeConfig, discovered)
	if err != nil {
		t.Fatalf("MergeKubeconfigs failed: %v", err)
	}
	if _, ok := merged.Clusters["test-cluster"]; !ok {
		t.Error("expected test-cluster in merged config")
	}
}

func TestResolveMergeFilesEmptySourceDir(t *testing.T) {
	_, err := resolveMergeFiles(MergeConfig{SourceDir: ""})
	if err == nil {
		t.Fatal("expected error for empty source dir, got nil")
	}
}

func TestResolveMergeFilesNotADirectory(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "not-a-dir")
	if err := os.WriteFile(filePath, []byte("test"), 0600); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	_, err := resolveMergeFiles(MergeConfig{
		SourceDir:       filePath,
		IncludePatterns: []string{"*.yaml"},
	})
	if err == nil {
		t.Fatal("expected error for non-directory source, got nil")
	}
}

func TestMergeIntoNilTargetOrSource(t *testing.T) {
	t.Run("nil target does not panic", func(_ *testing.T) {
		source := newKubeconfig()
		mergeInto(nil, source)
	})

	t.Run("nil source does not panic", func(_ *testing.T) {
		target := newKubeconfig()
		mergeInto(target, nil)
	})
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o700)
}
