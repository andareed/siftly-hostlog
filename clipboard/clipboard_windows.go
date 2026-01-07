// clipboard/clipboard_windows.go
//go:build windows
// +build windows

package clipboard

import (
	"os/exec"
	"strings"
)

// Copy uses PowerShell's Set-Clipboard to copy text on Windows.
// TODO: Replace Windows and Mac methods with attoto/clipboard library

func Copy(text string) error {
	cmd := exec.Command("powershell", "-NoProfile", "-Command", "Set-Clipboard")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
