//go:build unix

package terminal

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/term"
)

// TerminalState represents the state of a terminal
type TerminalState struct {
	fd    int
	state *term.State
}

// GetState gets the current terminal state
func GetState(fd int) (*TerminalState, error) {
	state, err := term.GetState(fd)
	if err != nil {
		return nil, err
	}
	return &TerminalState{
		fd:    fd,
		state: state,
	}, nil
}

// Restore restores the terminal to its previous state
func (ts *TerminalState) Restore() error {
	if ts.state == nil {
		// Log warning but don't return error for nil state
		if IsTerminal(ts.fd) {
			// Only log if it's actually a terminal
			fmt.Fprintf(os.Stderr, "Warning: terminal state is nil, cannot restore\n")
		}
		return nil
	}
	return term.Restore(ts.fd, ts.state)
}

// MakeRaw puts the terminal into raw mode
func MakeRaw(fd int) (*TerminalState, error) {
	oldState, err := term.GetState(fd)
	if err != nil {
		return nil, err
	}

	if _, err := term.MakeRaw(fd); err != nil {
		return nil, err
	}

	return &TerminalState{
		fd:    fd,
		state: oldState,
	}, nil
}

// SetupTerminal sets up the terminal for interactive mode
func SetupTerminal() (*TerminalState, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return nil, nil
	}

	return GetState(fd)
}

// IsTerminal checks if the given file descriptor is a terminal
func IsTerminal(fd int) bool {
	return term.IsTerminal(fd)
}

// GetSize returns the dimensions of the terminal
func GetSize(fd int) (width, height int, err error) {
	return term.GetSize(fd)
}

// SetWindowSize sets the terminal window size (Unix only)
func SetWindowSize(fd int, width, height int) error {
	ws := &winsize{
		Row: uint16(height),
		Col: uint16(width),
	}
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(syscall.TIOCSWINSZ),
		uintptr(unsafe.Pointer(ws)),
	)
	if errno != 0 {
		return errno
	}
	return nil
}

type winsize struct {
	Row uint16
	Col uint16
	X   uint16
	Y   uint16
}
