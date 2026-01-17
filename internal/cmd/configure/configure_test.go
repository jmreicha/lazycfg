package configure

import (
	"os"
	"testing"
)

func TestRunConfiguration(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}

	stdin := os.Stdin
	defer func() {
		os.Stdin = stdin
	}()

	os.Stdin = reader

	if _, err := writer.WriteString("value\n"); err != nil {
		_ = writer.Close()
		t.Fatalf("failed to write input: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

	if err := RunConfiguration(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
