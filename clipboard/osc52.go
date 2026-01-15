package clipboard

import (
	"encoding/base64"
	"errors"
	"os"
	"strings"

	"github.com/andareed/siftly-hostlog/logging"
)

func copyOSC52(text string) error {
	if !osc52Supported() {
		logging.Warnf("Clipboard: OSC52 unavailable (stdout not TTY or TERM=dumb)")
		return errors.New("clipboard unavailable (OSC52 unsupported by terminal)")
	}

	encoded := base64.StdEncoding.EncodeToString([]byte(text))
	seq := "\x1b]52;c;" + encoded + "\x07"
	if _, err := os.Stdout.WriteString(seq); err != nil {
		logging.Warnf("Clipboard: OSC52 write failed: %v", err)
		return err
	}
	logging.Infof("Clipboard: copied via OSC52")
	return nil
}

func osc52Supported() bool {
	if term := os.Getenv("TERM"); term == "" || strings.EqualFold(term, "dumb") {
		return false
	}
	return isTTY(os.Stdout)
}

func isTTY(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
