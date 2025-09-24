package scripting

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"
	"encoding/hex"
	"encoding/base64"
	"crypto/tls"

	"github.com/ibrahmsql/gocat/internal/logger"
	lua "github.com/yuin/gopher-lua"
)

// LuaEngine represents a Lua scripting engine for GoCat
type LuaEngine struct {
	L        *lua.LState
	config   *EngineConfig
	loadedScripts map[string]bool
}

// EngineConfig holds configuration for the Lua engine
type EngineConfig struct {
	MaxExecutionTime time.Duration
	MaxMemory        int64
	RestrictedMode   bool
	AllowedHosts     []string
	DeniedHosts      []string
}

// NewLuaEngine creates a new Lua engine with optional configuration
func NewLuaEngine(config *EngineConfig) *LuaEngine {
	if config == nil {
		config = &EngineConfig{
			MaxExecutionTime: 30 * time.Second,
			MaxMemory:        64 * 1024 * 1024, // 64MB
			RestrictedMode:   false,
		}
	}

	L := lua.NewState()
	
	// Configure Lua state limits
	L.SetMx(int(config.MaxMemory))
	
	engine := &LuaEngine{
		L:             L,
		config:        config,
		loadedScripts: make(map[string]bool),
	}

	// Register GoCat API functions
	engine.registerAPI()

	return engine
}

// Close closes the Lua engine and releases resources
func (e *LuaEngine) Close() {
	if e.L != nil {
		e.L.Close()
		e.L = nil
	}
}

// LoadScript loads a Lua script from file
func (e *LuaEngine) LoadScript(scriptPath string) error {
	if e.L == nil {
		return fmt.Errorf("lua engine is closed")
	}

	// Check if script already loaded
	if e.loadedScripts[scriptPath] {
		logger.Debug("Script already loaded: %s", scriptPath)
		return nil
	}

	// Read script file
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("failed to read script file: %w", err)
	}

	// Compile and load script
	if err := e.L.DoString(string(content)); err != nil {
		return fmt.Errorf("failed to load script: %w", err)
	}

	e.loadedScripts[scriptPath] = true
	logger.Debug("Successfully loaded script: %s", scriptPath)
	return nil
}

// ExecuteScript executes a loaded script's main function
func (e *LuaEngine) ExecuteScript(scriptName string) error {
	if e.L == nil {
		return fmt.Errorf("lua engine is closed")
	}

	// Execute main script (already loaded via DoString)
	logger.Info("Executing script: %s", scriptName)
	return nil
}

// ExecuteFunction executes a specific function in the loaded script
func (e *LuaEngine) ExecuteFunction(functionName string, args ...lua.LValue) ([]lua.LValue, error) {
	if e.L == nil {
		return nil, fmt.Errorf("lua engine is closed")
	}

	// Get the function
	fn := e.L.GetGlobal(functionName)
	if fn.Type() != lua.LTFunction {
		return nil, fmt.Errorf("function '%s' not found or not a function", functionName)
	}

	// Call function with arguments
	err := e.L.CallByParam(lua.P{
		Fn:      fn,
		NRet:    lua.MultRet,
		Protect: true,
	}, args...)

	if err != nil {
		return nil, fmt.Errorf("error executing function '%s': %w", functionName, err)
	}

	// Collect return values
	var results []lua.LValue
	top := e.L.GetTop()
	for i := 1; i <= top; i++ {
		results = append(results, e.L.Get(i))
	}
	e.L.SetTop(0) // Clear stack

	return results, nil
}

// registerAPI registers GoCat's Lua API functions
func (e *LuaEngine) registerAPI() {
	// Network functions
	e.L.SetGlobal("connect", e.L.NewFunction(e.luaConnect))
	e.L.SetGlobal("listen", e.L.NewFunction(e.luaListen))
	e.L.SetGlobal("send", e.L.NewFunction(e.luaSend))
	e.L.SetGlobal("receive", e.L.NewFunction(e.luaReceive))
	e.L.SetGlobal("close", e.L.NewFunction(e.luaClose))

	// Utility functions
	e.L.SetGlobal("log", e.L.NewFunction(e.luaLog))
	e.L.SetGlobal("sleep", e.L.NewFunction(e.luaSleep))
	e.L.SetGlobal("hex_encode", e.L.NewFunction(e.luaHexEncode))
	e.L.SetGlobal("hex_decode", e.L.NewFunction(e.luaHexDecode))
	e.L.SetGlobal("base64_encode", e.L.NewFunction(e.luaBase64Encode))
	e.L.SetGlobal("base64_decode", e.L.NewFunction(e.luaBase64Decode))

	// GoCat environment info
	gocatTable := e.L.NewTable()
	gocatTable.RawSetString("version", lua.LString("dev"))
	gocatTable.RawSetString("platform", lua.LString("linux/amd64"))
	e.L.SetGlobal("gocat", gocatTable)

	logger.Debug("Registered GoCat Lua API functions")
}

// Lua API function implementations

// luaConnect implements connect(host, port, protocol) function
func (e *LuaEngine) luaConnect(L *lua.LState) int {
	host := L.ToString(1)
	port := L.ToInt(2)
	protocol := L.ToString(3)

	if host == "" || port <= 0 {
		L.Push(lua.LNil)
		L.Push(lua.LString("invalid host or port"))
		return 2
	}

	// Check access control
	if e.config.RestrictedMode && !e.isHostAllowed(host) {
		L.Push(lua.LNil)
		L.Push(lua.LString("host not allowed in restricted mode"))
		return 2
	}

	address := fmt.Sprintf("%s:%d", host, port)

	var conn net.Conn
	var err error

	switch strings.ToLower(protocol) {
	case "tcp", "":
		conn, err = net.DialTimeout("tcp", address, 10*time.Second)
	case "udp":
		conn, err = net.DialTimeout("udp", address, 10*time.Second)
	case "ssl", "tls":
		config := &tls.Config{InsecureSkipVerify: true}
		conn, err = tls.DialWithDialer(&net.Dialer{Timeout: 10*time.Second}, "tcp", address, config)
	default:
		L.Push(lua.LNil)
		L.Push(lua.LString("unsupported protocol: " + protocol))
		return 2
	}

	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	// Store connection in userdata
	userData := L.NewUserData()
	userData.Value = conn
	L.Push(userData)
	L.Push(lua.LNil)
	return 2
}

// luaListen implements listen(port, protocol) function
func (e *LuaEngine) luaListen(L *lua.LState) int {
	port := L.ToInt(1)
	protocol := L.ToString(2)

	if port <= 0 || port > 65535 {
		L.Push(lua.LNil)
		L.Push(lua.LString("invalid port number"))
		return 2
	}

	address := fmt.Sprintf(":%d", port)

	var listener net.Listener
	var err error

	switch strings.ToLower(protocol) {
	case "tcp", "":
		listener, err = net.Listen("tcp", address)
	case "udp":
		// UDP doesn't have listeners in the traditional sense
		udpAddr, parseErr := net.ResolveUDPAddr("udp", address)
		if parseErr != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(parseErr.Error()))
			return 2
		}
		conn, udpErr := net.ListenUDP("udp", udpAddr)
		if udpErr != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(udpErr.Error()))
			return 2
		}
		userData := L.NewUserData()
		userData.Value = conn
		L.Push(userData)
		L.Push(lua.LNil)
		return 2
	default:
		L.Push(lua.LNil)
		L.Push(lua.LString("unsupported protocol: " + protocol))
		return 2
	}

	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	// Store listener in userdata
	userData := L.NewUserData()
	userData.Value = listener
	L.Push(userData)
	L.Push(lua.LNil)
	return 2
}

// luaSend implements send(conn, data) function
func (e *LuaEngine) luaSend(L *lua.LState) int {
	userData := L.ToUserData(1)
	data := L.ToString(2)

	if userData == nil || userData.Value == nil {
		L.Push(lua.LNumber(0))
		L.Push(lua.LString("invalid connection"))
		return 2
	}

	conn, ok := userData.Value.(net.Conn)
	if !ok {
		L.Push(lua.LNumber(0))
		L.Push(lua.LString("invalid connection type"))
		return 2
	}

	n, err := conn.Write([]byte(data))
	if err != nil {
		L.Push(lua.LNumber(0))
		L.Push(lua.LString(err.Error()))
		return 2
	}

	L.Push(lua.LNumber(n))
	L.Push(lua.LNil)
	return 2
}

// luaReceive implements receive(conn, size) function
func (e *LuaEngine) luaReceive(L *lua.LState) int {
	userData := L.ToUserData(1)
	size := L.ToInt(2)

	if userData == nil || userData.Value == nil {
		L.Push(lua.LString(""))
		L.Push(lua.LString("invalid connection"))
		return 2
	}

	conn, ok := userData.Value.(net.Conn)
	if !ok {
		L.Push(lua.LString(""))
		L.Push(lua.LString("invalid connection type"))
		return 2
	}

	if size <= 0 {
		size = 1024
	}

	buffer := make([]byte, size)
	n, err := conn.Read(buffer)
	if err != nil {
		L.Push(lua.LString(""))
		L.Push(lua.LString(err.Error()))
		return 2
	}

	L.Push(lua.LString(string(buffer[:n])))
	L.Push(lua.LNil)
	return 2
}

// luaClose implements close(conn) function
func (e *LuaEngine) luaClose(L *lua.LState) int {
	userData := L.ToUserData(1)

	if userData == nil || userData.Value == nil {
		L.Push(lua.LBool(false))
		return 1
	}

	if conn, ok := userData.Value.(net.Conn); ok {
		conn.Close()
		L.Push(lua.LBool(true))
		return 1
	}

	if listener, ok := userData.Value.(net.Listener); ok {
		listener.Close()
		L.Push(lua.LBool(true))
		return 1
	}

	L.Push(lua.LBool(false))
	return 1
}

// luaLog implements log(level, message) function
func (e *LuaEngine) luaLog(L *lua.LState) int {
	level := L.ToString(1)
	message := L.ToString(2)

	switch strings.ToLower(level) {
	case "debug":
		logger.Debug("[Script] %s", message)
	case "info":
		logger.Info("[Script] %s", message)
	case "warn", "warning":
		logger.Warn("[Script] %s", message)
	case "error":
		logger.Error("[Script] %s", message)
	default:
		logger.Info("[Script] %s", message)
	}

	return 0
}

// luaSleep implements sleep(seconds) function
func (e *LuaEngine) luaSleep(L *lua.LState) int {
	seconds := L.ToNumber(1)
	if seconds > 0 {
		time.Sleep(time.Duration(float64(seconds) * float64(time.Second)))
	}
	return 0
}

// luaHexEncode implements hex_encode(data) function
func (e *LuaEngine) luaHexEncode(L *lua.LState) int {
	data := L.ToString(1)
	encoded := hex.EncodeToString([]byte(data))
	L.Push(lua.LString(encoded))
	return 1
}

// luaHexDecode implements hex_decode(hex) function
func (e *LuaEngine) luaHexDecode(L *lua.LState) int {
	hexStr := L.ToString(1)
	decoded, err := hex.DecodeString(hexStr)
	if err != nil {
		L.Push(lua.LString(""))
		return 1
	}
	L.Push(lua.LString(string(decoded)))
	return 1
}

// luaBase64Encode implements base64_encode(data) function
func (e *LuaEngine) luaBase64Encode(L *lua.LState) int {
	data := L.ToString(1)
	encoded := base64.StdEncoding.EncodeToString([]byte(data))
	L.Push(lua.LString(encoded))
	return 1
}

// luaBase64Decode implements base64_decode(b64) function
func (e *LuaEngine) luaBase64Decode(L *lua.LState) int {
	b64Str := L.ToString(1)
	decoded, err := base64.StdEncoding.DecodeString(b64Str)
	if err != nil {
		L.Push(lua.LString(""))
		return 1
	}
	L.Push(lua.LString(string(decoded)))
	return 1
}

// isHostAllowed checks if host is allowed for connection
func (e *LuaEngine) isHostAllowed(host string) bool {
	// If no restrictions, allow all
	if len(e.config.AllowedHosts) == 0 && len(e.config.DeniedHosts) == 0 {
		return true
	}

	// Check denied hosts first
	for _, deniedHost := range e.config.DeniedHosts {
		if strings.Contains(host, deniedHost) {
			return false
		}
	}

	// If no allowed hosts specified, allow all (except denied)
	if len(e.config.AllowedHosts) == 0 {
		return true
	}

	// Check allowed hosts
	for _, allowedHost := range e.config.AllowedHosts {
		if strings.Contains(host, allowedHost) {
			return true
		}
	}

	return false
}