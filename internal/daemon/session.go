package daemon

import (
	"os/exec"
	"strings"
)

// hasActiveSSHSession returns true if the user has at least one interactive
// outgoing SSH session open. It checks for "ssh" processes that have a
// controlling terminal (TTY), which excludes background/non-interactive calls
// like cpssh's own sync connections (those show "??" / "?" in the TTY column).
func hasActiveSSHSession() bool {
	out, err := exec.Command("ps", "-A", "-o", "tty,comm").Output()
	if err != nil {
		return true // fail open: can't check, assume session is active
	}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == "ssh" && fields[0] != "??" && fields[0] != "?" {
			return true
		}
	}
	return false
}
