package kubernetes

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

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

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o700)
}
