package kubernetes

import (
	"os"
	"path/filepath"
)

func readFixture(name string) ([]byte, error) {
	path := filepath.Join("testdata", name)
	// #nosec G304 -- path is limited to testdata fixtures.
	return os.ReadFile(path)
}

func writeFixture(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o600)
}
