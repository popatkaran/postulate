// Package browser opens URLs in the system browser.
// The exec function is injectable so tests can run without a real browser.
package browser

import (
	"fmt"
	"os/exec"
	"runtime"
)

// ExecFunc is the type of the function used to launch a subprocess.
// Defaults to exec.Command(...).Run(); injectable for testing.
type ExecFunc func(name string, args ...string) error

// defaultExec runs the command and waits for it to exit.
func defaultExec(name string, args ...string) error {
	return exec.Command(name, args...).Run() //nolint:gosec
}

// Open attempts to open url in the default system browser using fn.
// If fn is nil, defaultExec is used.
// Returns an error if the launch fails; callers should fall back to printing the URL.
func Open(url string, fn ExecFunc) error {
	if fn == nil {
		fn = defaultExec
	}
	switch runtime.GOOS {
	case "darwin":
		return fn("open", url)
	case "windows":
		return fn("cmd", "/c", "start", url)
	default: // linux and others
		return fn("xdg-open", url)
	}
}

// OpenOrPrint tries to open url in the browser. On failure it prints a manual
// instruction to stdout via printf.
func OpenOrPrint(url string, fn ExecFunc, printf func(format string, a ...any)) {
	if err := Open(url, fn); err != nil {
		printf("Could not open browser automatically.\nOpen this URL manually:\n\n  %s\n\n", url)
	}
}

// Sprintf is a convenience alias so callers can pass fmt.Printf directly.
var Sprintf = fmt.Sprintf
