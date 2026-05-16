package desktop

import (
	"fmt"
	"os/exec"
	"runtime"
)

// OpenBrowser launches the system default browser at url.
func OpenBrowser(url string) error {
	var name string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		name = "open"
		args = []string{url}
	case "windows":
		name = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	default:
		name = "xdg-open"
		args = []string{url}
	}
	if err := exec.Command(name, args...).Start(); err != nil {
		return fmt.Errorf("desktop: open browser: %w", err)
	}
	return nil
}
