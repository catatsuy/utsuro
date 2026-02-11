package term

import "os"

// IsTerminal reports whether the file descriptor seems to be a terminal.
func IsTerminal(fd int) bool {
	f := os.NewFile(uintptr(fd), "")
	if f == nil {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
