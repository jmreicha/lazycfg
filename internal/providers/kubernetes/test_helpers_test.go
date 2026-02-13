package kubernetes

import (
	"embed"
	"os"
	"path/filepath"
)

//go:embed testdata/*
var testdataFS embed.FS

func readFixture(name string) ([]byte, error) {
	path := filepath.Join("testdata", name)
	return testdataFS.ReadFile(path)
}

func writeFixture(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o600)
}
