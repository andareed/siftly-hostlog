//go:build darwin
// +build darwin

package clipboard

import (
	"os/exec"
	"strings"
)

// Copy uses pbcopy on macOS.
func Copy(text string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
