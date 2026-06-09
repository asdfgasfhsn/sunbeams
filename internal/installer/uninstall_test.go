package installer

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// TestUninstall_RequiresRoot mirrors TestRun_RequiresRoot: the only path
// exercisable without real system access is the root check. The rest of
// Uninstall shells out to rpm-ostree/systemctl and is validated on live Bazzite.
func TestUninstall_RequiresRoot(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root — root-check path is unreachable, skipping")
	}

	var stdout bytes.Buffer
	stdin := strings.NewReader("")

	err := Uninstall("", false, stdin, &stdout)
	if err == nil {
		t.Fatal("expected Uninstall to return an error when not root, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "root") {
		t.Errorf("error %q does not mention root", err.Error())
	}
	if stdout.Len() != 0 {
		t.Errorf("expected no output before root check, got: %q", stdout.String())
	}
}
