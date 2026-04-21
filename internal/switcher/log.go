package switcher

import (
	"fmt"
	"os"
	"time"
)

// logf writes a timestamped [sunbeams LEVEL HH:MM:SS] line to stderr so it
// interleaves cleanly with Sunshine's own logging and stays out of stdout
// (which some callers may parse).
func logf(level, format string, args ...any) {
	ts := time.Now().Format("15:04:05")
	fmt.Fprintf(os.Stderr, "[sunbeams %s %s] %s\n", level, ts, fmt.Sprintf(format, args...))
}

func info(format string, args ...any)   { logf("info ", format, args...) }
func warn(format string, args ...any)   { logf("warn ", format, args...) }
func errLog(format string, args ...any) { logf("error", format, args...) }

func debugEnabled() bool {
	v := os.Getenv("SUNBEAMS_DEBUG")
	return v == "1" || v == "true"
}

func debug(format string, args ...any) {
	if debugEnabled() {
		logf("debug", format, args...)
	}
}
