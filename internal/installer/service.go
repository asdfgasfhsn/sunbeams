package installer

import "fmt"

func UserServiceUnit(execPath string) string {
	return fmt.Sprintf(`[Unit]
Description=Add custom display modes for Moonlight streaming
After=graphical-session.target

[Service]
Type=oneshot
ExecStart=%s
Environment=DISPLAY=:0

[Install]
WantedBy=graphical-session.target
`, execPath)
}
