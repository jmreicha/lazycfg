package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// FindExecutable returns the path to an executable if found.
func FindExecutable(name string) (string, bool) {
	if findExecutableHook != nil {
		return findExecutableHook(name)
	}

	path, err := exec.LookPath(name)
	if err == nil {
		return path, true
	}

	if runtime.GOOS == "windows" {
		return "", false
	}

	for _, dir := range commonExecutablePaths() {
		candidate := filepath.Join(dir, name)
		if isExecutable(candidate) {
			return candidate, true
		}
	}

	return "", false
}

var findExecutableHook func(name string) (string, bool)

// MissingExecutables returns names not found on the system.
func MissingExecutables(names ...string) []string {
	missing := make([]string, 0, len(names))
	for _, name := range names {
		if _, ok := FindExecutable(name); !ok {
			missing = append(missing, name)
		}
	}

	return missing
}

func commonExecutablePaths() []string {
	if runtime.GOOS == "darwin" {
		return []string{
			"/opt/homebrew/bin",
			"/usr/local/bin",
			"/usr/bin",
			"/bin",
			"/opt/local/bin",
			"/usr/sbin",
			"/sbin",
		}
	}

	return []string{
		"/usr/local/bin",
		"/usr/bin",
		"/bin",
		"/usr/sbin",
		"/sbin",
	}
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}

	return info.Mode()&0111 != 0
}
