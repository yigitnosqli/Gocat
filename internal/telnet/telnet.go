package telnet

import (
	"fmt"
	"io"
	"net"

	"github.com/ibrahmsql/gocat/internal/logger"
)

// Telnet command codes
const (
	IAC  = 255 // Interpret As Command
	DONT = 254 // Don't use option
	DO   = 253 // Use option
	WONT = 252 // Won't use option
	WILL = 251 // Will use option
	SB   = 250 // Subnegotiation Begin
	SE   = 240 // Subnegotiation End
	
	// Telnet options
	ECHO                 = 1   // Echo
	SUPPRESS_GO_AHEAD    = 3   // Suppress Go Ahead
	STATUS               = 5   // Status
	TIMING_MARK          = 6   // Timing Mark
	TERMINAL_TYPE        = 24  // Terminal Type
	WINDOW_SIZE          = 31  // Window Size
	TERMINAL_SPEED       = 32  // Terminal Speed
	REMOTE_FLOW_CONTROL  = 33  // Remote Flow Control
	LINEMODE             = 34  // Linemode
	ENVIRONMENT_VARIABLES = 36 // Environment Variables
)

// TelnetHandler handles Telnet protocol negotiations
type TelnetHandler struct {
	conn   net.Conn
	reader io.Reader
	writer io.Writer
	debug  bool
}

// NewTelnetHandler creates a new Telnet handler
func NewTelnetHandler(conn net.Conn) *TelnetHandler {
	return &TelnetHandler{
		conn:   conn,
		reader: conn,
		writer: conn,
		debug:  false,
	}
}

// SetDebug enables or disables debug logging
func (th *TelnetHandler) SetDebug(debug bool) {
	th.debug = debug
}

// HandleNegotiation processes Telnet negotiations
func (th *TelnetHandler) HandleNegotiation() error {
	buffer := make([]byte, 1024)
	
	for {
		n, err := th.reader.Read(buffer)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("read error: %v", err)
		}
		
		data := buffer[:n]
		processed, err := th.processData(data)
		if err != nil {
			return fmt.Errorf("processing error: %v", err)
		}
		
		if len(processed) > 0 {
			// Return processed data to caller
			// This would typically be handled by a callback or channel
			if th.debug {
				logger.Debug("Processed data: %q", string(processed))
			}
		}
	}
}

// processData processes incoming data and handles Telnet commands
func (th *TelnetHandler) processData(data []byte) ([]byte, error) {
	var result []byte
	i := 0
	
	for i < len(data) {
		if data[i] == IAC && i+1 < len(data) {
			// Handle Telnet command
			cmd := data[i+1]
			
			switch cmd {
			case IAC:
				// Escaped IAC, add single IAC to result
				result = append(result, IAC)
				i += 2
				
			case DO, DONT, WILL, WONT:
				if i+2 < len(data) {
					option := data[i+2]
					err := th.handleOption(cmd, option)
					if err != nil {
						return result, err
					}
					i += 3
				} else {
					// Incomplete command, need more data
					return result, nil
				}
				
			case SB:
				// Subnegotiation - find SE
				end := th.findSubnegotiationEnd(data[i:])
				if end == -1 {
					// Incomplete subnegotiation
					return result, nil
				}
				
				err := th.handleSubnegotiation(data[i:i+end+1])
				if err != nil {
					return result, err
				}
				i += end + 1
				
			default:
				if th.debug {
					logger.Debug("Unknown Telnet command: %d", cmd)
				}
				i += 2
			}
		} else {
			// Regular data
			result = append(result, data[i])
			i++
		}
	}
	
	return result, nil
}

// handleOption handles Telnet option negotiations
func (th *TelnetHandler) handleOption(cmd, option byte) error {
	if th.debug {
		logger.Debug("Telnet option: %s %s", th.commandName(cmd), th.optionName(option))
	}
	
	switch cmd {
	case DO:
		// Server wants us to enable an option
		switch option {
		case ECHO, SUPPRESS_GO_AHEAD:
			// We'll handle these options
			return th.sendResponse(WILL, option)
		default:
			// We won't handle other options
			return th.sendResponse(WONT, option)
		}
		
	case DONT:
		// Server wants us to disable an option
		return th.sendResponse(WONT, option)
		
	case WILL:
		// Server will enable an option
		switch option {
		case ECHO, SUPPRESS_GO_AHEAD:
			// We accept these options
			return th.sendResponse(DO, option)
		default:
			// We don't want other options
			return th.sendResponse(DONT, option)
		}
		
	case WONT:
		// Server won't enable an option
		return th.sendResponse(DONT, option)
	}
	
	return nil
}

// handleSubnegotiation handles Telnet subnegotiations
func (th *TelnetHandler) handleSubnegotiation(data []byte) error {
	if len(data) < 4 {
		return fmt.Errorf("invalid subnegotiation data")
	}
	
	option := data[2]
	subData := data[3 : len(data)-2] // Remove IAC SB ... IAC SE
	
	if th.debug {
		logger.Debug("Telnet subnegotiation for option %s: %v", th.optionName(option), subData)
	}
	
	switch option {
	case TERMINAL_TYPE:
		return th.handleTerminalType(subData)
	case WINDOW_SIZE:
		return th.handleWindowSize(subData)
	case ENVIRONMENT_VARIABLES:
		return th.handleEnvironment(subData)
	default:
		if th.debug {
			logger.Debug("Unhandled subnegotiation for option %d", option)
		}
	}
	
	return nil
}

// handleTerminalType handles terminal type subnegotiation
func (th *TelnetHandler) handleTerminalType(data []byte) error {
	if len(data) > 0 && data[0] == 1 { // SEND
		// Send our terminal type
		response := []byte{IAC, SB, TERMINAL_TYPE, 0} // IS
		response = append(response, []byte("xterm")...)
		response = append(response, IAC, SE)
		
		_, err := th.writer.Write(response)
		return err
	}
	return nil
}

// handleWindowSize handles window size subnegotiation
func (th *TelnetHandler) handleWindowSize(data []byte) error {
	if len(data) >= 4 {
		width := (int(data[0]) << 8) | int(data[1])
		height := (int(data[2]) << 8) | int(data[3])
		
		if th.debug {
			logger.Debug("Window size: %dx%d", width, height)
		}
	}
	return nil
}

// handleEnvironment handles environment variable subnegotiation
func (th *TelnetHandler) handleEnvironment(data []byte) error {
	if th.debug {
		logger.Debug("Environment variables: %v", data)
	}
	return nil
}

// sendResponse sends a Telnet response
func (th *TelnetHandler) sendResponse(cmd, option byte) error {
	response := []byte{IAC, cmd, option}
	_, err := th.writer.Write(response)
	
	if th.debug {
		logger.Debug("Sent Telnet response: %s %s", th.commandName(cmd), th.optionName(option))
	}
	
	return err
}

// findSubnegotiationEnd finds the end of a subnegotiation sequence
func (th *TelnetHandler) findSubnegotiationEnd(data []byte) int {
	for i := 0; i < len(data)-1; i++ {
		if data[i] == IAC && data[i+1] == SE {
			return i + 1
		}
	}
	return -1
}

// commandName returns the name of a Telnet command
func (th *TelnetHandler) commandName(cmd byte) string {
	switch cmd {
	case DO:
		return "DO"
	case DONT:
		return "DONT"
	case WILL:
		return "WILL"
	case WONT:
		return "WONT"
	case SB:
		return "SB"
	case SE:
		return "SE"
	default:
		return fmt.Sprintf("CMD_%d", cmd)
	}
}

// optionName returns the name of a Telnet option
func (th *TelnetHandler) optionName(option byte) string {
	switch option {
	case ECHO:
		return "ECHO"
	case SUPPRESS_GO_AHEAD:
		return "SUPPRESS_GO_AHEAD"
	case STATUS:
		return "STATUS"
	case TIMING_MARK:
		return "TIMING_MARK"
	case TERMINAL_TYPE:
		return "TERMINAL_TYPE"
	case WINDOW_SIZE:
		return "WINDOW_SIZE"
	case TERMINAL_SPEED:
		return "TERMINAL_SPEED"
	case REMOTE_FLOW_CONTROL:
		return "REMOTE_FLOW_CONTROL"
	case LINEMODE:
		return "LINEMODE"
	case ENVIRONMENT_VARIABLES:
		return "ENVIRONMENT_VARIABLES"
	default:
		return fmt.Sprintf("OPTION_%d", option)
	}
}

// WrapConnection wraps a connection with Telnet protocol handling
func WrapConnection(conn net.Conn, enableNegotiation bool) io.ReadWriteCloser {
	if !enableNegotiation {
		return conn
	}
	
	return &TelnetConnection{
		conn:    conn,
		handler: NewTelnetHandler(conn),
	}
}

// TelnetConnection wraps a connection with Telnet handling
type TelnetConnection struct {
	conn          net.Conn
	handler       *TelnetHandler
	readBuffer    []byte
	processedData []byte
}

// Read implements io.Reader with Telnet processing
func (tc *TelnetConnection) Read(p []byte) (n int, err error) {
	// Buffer for processing Telnet commands
	if tc.readBuffer == nil {
		tc.readBuffer = make([]byte, 0, 4096)
	}

	// If we have processed data in buffer, return it first
	if len(tc.processedData) > 0 {
		n = copy(p, tc.processedData)
		tc.processedData = tc.processedData[n:]
		return n, nil
	}

	// Read raw data from connection
	rawBuf := make([]byte, len(p))
	rawN, err := tc.conn.Read(rawBuf)
	if err != nil {
		return 0, err
	}

	// Process Telnet commands and extract clean data
	processed, procErr := tc.handler.processData(rawBuf[:rawN])
	if procErr != nil {
		return 0, procErr
	}

	// Copy processed data to output buffer
	n = copy(p, processed)
	
	// Store remaining processed data for next read
	if len(processed) > n {
		tc.processedData = append(tc.processedData, processed[n:]...)
	}

	return n, nil
}

// Write implements io.Writer
func (tc *TelnetConnection) Write(p []byte) (n int, err error) {
	return tc.conn.Write(p)
}

// Close implements io.Closer
func (tc *TelnetConnection) Close() error {
	return tc.conn.Close()
}