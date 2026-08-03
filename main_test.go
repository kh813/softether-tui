package main

import (
	"os/exec"
	"strings"
	"testing"
)

// TestVersionOutputBuildWithLdflags verifies that binaries built via Makefile
// or go build with ldflags correctly output version, commit hash, and build timestamp.
func TestVersionOutputBuildWithLdflags(t *testing.T) {
	cmd := exec.Command("make", "build", "VERSION=v0.1.0-test")
	cmd.Dir = "."
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("make build failed: %v\nOutput: %s", err, string(out))
	}

	outCmd := exec.Command("./softether-tui", "--version")
	output, err := outCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("./softether-tui --version failed: %v", err)
	}

	outStr := strings.TrimSpace(string(output))
	if !strings.Contains(outStr, "v0.1.0-test") {
		t.Errorf("expected version output to contain 'v0.1.0-test', got: %s", outStr)
	}
	if strings.Contains(outStr, "commit none") {
		t.Errorf("expected version output to contain git commit hash instead of 'commit none', got: %s", outStr)
	}
	if strings.Contains(outStr, "built unknown") {
		t.Errorf("expected version output to contain build date instead of 'built unknown', got: %s", outStr)
	}
}
