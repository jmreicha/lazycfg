package tool

import (
	"fmt"
	"os"
)

// RunConfiguration creates a file at /tmp/test-config containing "Hello World".
// It returns an error if the file operation fails.
func CreateToolConfiguration() error {
	filePath := "/tmp/test-config"

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, writeErr := file.WriteString("Hello World\n")
	if writeErr != nil {
		return fmt.Errorf("failed to write to file: %w", writeErr)
	}

	return nil
}
