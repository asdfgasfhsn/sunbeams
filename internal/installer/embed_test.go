package installer

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHelperScript_NotEmpty(t *testing.T) {
	s := HelperScript()
	assert.NotEmpty(t, s, "embedded helper must not be empty")
}

func TestHelperScript_HasShebang(t *testing.T) {
	s := HelperScript()
	assert.True(t, bytes.HasPrefix(s, []byte("#!/bin/bash")), "helper must start with bash shebang")
}

func TestHelperScript_RejectsBadConnectorPattern(t *testing.T) {
	// We can't actually exec it on macOS without bash + the debugfs files.
	// Just check that the source contains the validation regex covering DP-1.
	s := string(HelperScript())
	assert.Contains(t, s, `^[A-Za-z]+(-[A-Z])?-[0-9]+$`, "regex must accept DP-1 / eDP-1 / HDMI-A-1")
	assert.Contains(t, s, "exit 2", "must exit on validation failure")
	assert.Contains(t, s, "exit 3", "must exit on debugfs absence/ambiguity")
}

func TestHelperScript_HasShellcheckSafetyFlags(t *testing.T) {
	s := string(HelperScript())
	assert.Contains(t, s, "set -euo pipefail")
	// shopt -s nullglob is critical so the glob doesn't expand to itself when empty.
	assert.True(t, strings.Contains(s, "shopt -s nullglob"))
}
