//go:build linux
// +build linux

package clipboard

import (
	"errors"
	"os"
	"os/exec"
	"strings"
)

// Copy copies text to the Wayland clipboard using wl-copy.
// This intentionally does NOT support X11. Wayland or bust, sorry n all
func Copy(text string) error {
	// Require some sign of Wayland
	if os.Getenv("WAYLAND_DISPLAY") == "" && os.Getenv("XDG_SESSION_TYPE") != "wayland" {
		return errors.New("wayland clipboard not available (no WAYLAND_DISPLAY and XDG_SESSION_TYPE!=wayland)")
	}

	if _, err := exec.LookPath("wl-copy"); err != nil {
		return errors.New("wl-copy not found in PATH (install wl-clipboard)")
	}

	cmd := exec.Command("wl-copy")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
