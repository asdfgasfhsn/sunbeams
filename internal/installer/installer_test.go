package installer

import (
	"bytes"
	"strings"
	"testing"

	"os"
)

// TestRun_RequiresRoot verifies that Run returns an error containing "root"
// when the process is not running as root. This is the only path exercisable
// in a normal CI environment without real system access.
//
// Coverage note: this test covers ONE early-return path. The remainder of Run
// (EDID write, connector scan, kargs injection, user-service install) still
// lacks unit coverage and is validated by live-Bazzite integration testing.
func TestRun_RequiresRoot(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root — root-check path is unreachable, skipping")
	}

	var stdout bytes.Buffer
	stdin := strings.NewReader("")

	err := Run(Options{
		EDIDBytes:   []byte("fake-edid"),
		ModesScript: []byte("fake-modes-script"),
		Stdin:       stdin,
		Stdout:      &stdout,
		Gaming:      GamingNo, // EDID-only test, skip gaming block
	})

	if err == nil {
		t.Fatal("expected Run to return an error when not root, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "root") {
		t.Errorf("error %q does not mention %q", err.Error(), "root")
	}
	if stdout.Len() != 0 {
		t.Errorf("expected no output on stdout before root check, got: %q", stdout.String())
	}
}
