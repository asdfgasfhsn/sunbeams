package installer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserServiceUnit(t *testing.T) {
	got := UserServiceUnit("/home/user/.local/bin/add-virtual-display-modes.sh")
	assert.Contains(t, got, "Description=Add custom display modes for Moonlight streaming")
	assert.Contains(t, got, "ExecStart=/home/user/.local/bin/add-virtual-display-modes.sh")
	assert.Contains(t, got, "WantedBy=graphical-session.target")
}
