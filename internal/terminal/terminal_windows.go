//go:build windows

package terminal

import (
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/term"
)

// TerminalState represents the state of a terminal on Windows
type TerminalState struct {
	fd    int
	state *term.State
	mode  uint32
}

const (
	ENABLE_VIRTUAL_TERMINAL_PROCESSING = 0x0004
	ENABLE_PROCESSED_OUTPUT            = 0x0001
	ENABLE_WRAP_AT_EOL_OUTPUT          = 0x0002
	DISABLE_NEWLINE_AUTO_RETURN        = 0x0008
)

// GetState gets the current terminal state (Windows)
func GetState(fd int) (*TerminalState, error) {
	state, err := term.GetState(fd)
	if err != nil {
		return nil, err
	}

	handle := syscall.Handle(fd)
	var mode uint32
	err2 := syscall.Syscall(procGetConsoleMode.Addr(), 2, uintptr(handle), uintptr(unsafe.Pointer(&mode)), 0)
	if err2 != 0 {
		return nil, syscall.Errno(err2)
	}

	return &TerminalState{
		fd:    fd,
		state: state,
		mode:  mode,
	}, nil
}

// Restore restores the terminal to its previous state (Windows)
func (ts *TerminalState) Restore() error {
	if ts.state != nil {
		if err := term.Restore(ts.fd, ts.state); err != nil {
			return err
		}
	}

	handle := syscall.Handle(ts.fd)
	err := syscall.Syscall(procSetConsoleMode.Addr(), 2, uintptr(handle), uintptr(ts.mode), 0)
	if err != 0 {
		return syscall.Errno(err)
	}
	return nil
}

// MakeRaw puts the terminal into raw mode (Windows)
func MakeRaw(fd int) (*TerminalState, error) {
	oldState, err := GetState(fd)
	if err != nil {
		return nil, err
	}

	if err := term.MakeRaw(fd); err != nil {
		return nil, err
	}

	// Enable virtual terminal processing for color support
	handle := syscall.Handle(fd)
	newMode := oldState.mode | ENABLE_VIRTUAL_TERMINAL_PROCESSING | ENABLE_PROCESSED_OUTPUT
	err2 := syscall.Syscall(procSetConsoleMode.Addr(), 2, uintptr(handle), uintptr(newMode), 0)
	if err2 != 0 {
		// Non-fatal error, continue without VT processing
	}

	return oldState, nil
}

// SetupTerminal sets up the terminal for interactive mode (Windows)
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

// SetWindowSize sets the terminal window size (Windows)
func SetWindowSize(fd int, width, height int) error {
	handle := syscall.Handle(fd)
	coord := coord{
		X: int16(width),
		Y: int16(height),
	}
	err := syscall.Syscall(procSetConsoleScreenBufferSize.Addr(), 2, uintptr(handle), uintptr(*((*uint32)(unsafe.Pointer(&coord)))), 0)
	if err != 0 {
		return syscall.Errno(err)
	}
	return nil
}

type coord struct {
	X int16
	Y int16
}

var (
	kernel32                       = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleMode             = kernel32.NewProc("GetConsoleMode")
	procSetConsoleMode             = kernel32.NewProc("SetConsoleMode")
	procSetConsoleScreenBufferSize = kernel32.NewProc("SetConsoleScreenBufferSize")
)
