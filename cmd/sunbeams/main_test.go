package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateE2E(t *testing.T) {
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "sunbeams")
	build := exec.Command("go", "build", "-o", bin, ".")
	out, err := build.CombinedOutput()
	require.NoError(t, err, "build failed: %s", out)

	run := exec.Command(bin, "generate", "--output-dir", tmp)
	out, err = run.CombinedOutput()
	require.NoError(t, err, "run failed: %s", out)

	got, err := os.ReadFile(filepath.Join(tmp, "virtual_display.bin"))
	require.NoError(t, err)
	want, err := os.ReadFile("../../testdata/virtual_display_reference.bin")
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestHelp_TopLevel(t *testing.T) {
	bin := buildBinary(t)
	for _, arg := range []string{"--help", "-h", "help"} {
		run := exec.Command(bin, arg)
		out, err := run.CombinedOutput()
		require.NoError(t, err, "exit non-zero for %q: %s", arg, out)
		s := string(out)
		for _, keyword := range []string{"USAGE:", "COMMANDS:", "generate", "switch", "install", "uninstall", "status", "TYPICAL WORKFLOW", "CONFIGURATION"} {
			assert.Contains(t, s, keyword, "top-level help for %q missing %q", arg, keyword)
		}
	}
}

func TestHelp_Subcommands(t *testing.T) {
	bin := buildBinary(t)
	cases := []struct {
		args            []string
		mustContain     []string
		mustHaveExample bool
	}{
		{[]string{"generate", "--help"}, []string{"USAGE:", "FLAGS:", "EXAMPLES:", "virtual_display.bin"}, true},
		{[]string{"switch", "--help"}, []string{"USAGE:", "SUBCOMMANDS:", "on", "off"}, false},
		{[]string{"switch", "on", "--help"}, []string{"USAGE:", "FLAGS:", "EXAMPLES:", "SUNSHINE_CLIENT"}, true},
		{[]string{"switch", "off", "--help"}, []string{"USAGE:", "EXAMPLES:"}, true},
		{[]string{"config", "--help"}, []string{"USAGE:", "SUBCOMMANDS:", "init", "show"}, false},
		{[]string{"config", "show", "--help"}, []string{"USAGE:", "FLAGS:", "EXAMPLES:"}, true},
		{[]string{"devices", "--help"}, []string{"USAGE:", "EXAMPLES:"}, true},
		{[]string{"modes", "--help"}, []string{"USAGE:", "DESCRIPTION:", "EXAMPLES:", "655 MHz"}, true},
		{[]string{"install", "--help"}, []string{"USAGE:", "DESCRIPTION:", "EXAMPLES:", "rpm-ostree"}, true},
		{[]string{"uninstall", "--help"}, []string{"USAGE:", "DESCRIPTION:", "EXAMPLES:", "--connector"}, true},
		{[]string{"status", "--help"}, []string{"USAGE:", "DESCRIPTION:", "EXAMPLES:"}, true},
		{[]string{"version", "--help"}, []string{"USAGE:", "EXAMPLES:"}, true},
	}
	for _, tc := range cases {
		t.Run(strings.Join(tc.args, "_"), func(t *testing.T) {
			run := exec.Command(bin, tc.args...)
			out, err := run.CombinedOutput()
			require.NoError(t, err, "exit non-zero: %s", out)
			s := string(out)
			for _, keyword := range tc.mustContain {
				assert.Contains(t, s, keyword, "help missing %q in output: %s", keyword, s)
			}
		})
	}
}

func TestHelp_UnknownCommand(t *testing.T) {
	bin := buildBinary(t)
	run := exec.Command(bin, "nonsense")
	out, err := run.CombinedOutput()
	assert.Error(t, err, "expected non-zero exit")
	s := string(out)
	assert.Contains(t, s, "unknown command: nonsense")
	assert.Contains(t, s, "USAGE:", "should also print top-level help")
}

func TestUninstall_RejectsPositionalArg(t *testing.T) {
	err := runUninstall([]string{"HDMI-A-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected argument")
	assert.Contains(t, err.Error(), "--connector")
	// Must NOT have fallen through to the installer (which would report the root check).
	assert.NotContains(t, err.Error(), "root")
}

func buildBinary(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "sunbeams")
	build := exec.Command("go", "build", "-o", bin, ".")
	out, err := build.CombinedOutput()
	require.NoError(t, err, "build failed: %s", out)
	return bin
}

func TestValidate_SkipsWhenMissing(t *testing.T) {
	tmp := t.TempDir()

	// Build the binary into the tmp dir.
	bin := filepath.Join(tmp, "sunbeams")
	build := exec.Command("go", "build", "-o", bin, ".")
	out, err := build.CombinedOutput()
	require.NoError(t, err, "build failed: %s", out)

	// Run generate --validate with PATH set to an empty dir so edid-decode
	// cannot be found. We point --output-dir at tmp so there's a writable
	// location for the EDID.
	emptyDir := t.TempDir()
	run := exec.Command(bin, "generate", "--output-dir", tmp, "--validate")
	run.Env = append(os.Environ(), "PATH="+emptyDir)
	combined, err := run.CombinedOutput()
	stdout := string(combined)

	require.NoError(t, err, "command failed (exit non-zero); output: %s", stdout)
	assert.True(t, strings.Contains(stdout, "skipping validation"),
		"expected 'skipping validation' in output; got: %s", stdout)
}
