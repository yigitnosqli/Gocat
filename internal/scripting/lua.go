package scripting

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ibrahmsql/gocat/internal/logger"
	"github.com/yuin/gopher-lua"
)

// LuaEngine manages Lua script execution
type LuaEngine struct {
	vm       *lua.LState
	scripts  map[string]*LuaScript
	mutex    sync.RWMutex
	timeout  time.Duration
	maxMem   int64
	sandbox  bool
}

// LuaScript represents a loaded Lua script
type LuaScript struct {
	Name     string
	Path     string
	Content  string
	Function *lua.LFunction
	Loaded   time.Time
}

// LuaConfig holds Lua engine configuration
type LuaConfig struct {
	Timeout    time.Duration // Script execution timeout
	MaxMemory  int64         // Maximum memory usage (bytes)
	Sandbox    bool          // Enable sandboxing
	ScriptDirs []string      // Directories to search for scripts
}

// DefaultLuaConfig returns a LuaConfig populated with sensible defaults:
// a 30 second script timeout, 64 MiB max memory, sandboxing enabled, and
// default script search paths ("./scripts" and "~/.gocat/scripts").
func DefaultLuaConfig() *LuaConfig {
	return &LuaConfig{
		Timeout:    30 * time.Second,
		MaxMemory:  64 * 1024 * 1024, // 64MB
		Sandbox:    true,
		ScriptDirs: []string{"./scripts", "~/.gocat/scripts"},
	}
}

// NewLuaEngine creates and initializes a LuaEngine using the provided configuration.
// If config is nil, DefaultLuaConfig() is used. The function creates a new Lua VM,
// applies the GoCat API and sandbox settings, and preloads any .lua scripts found in
// config.ScriptDirs. Returns an initialized *LuaEngine or an error if the Lua state
// cannot be created or the environment setup fails. Loading individual script files
// from directories is attempted but failures there are logged and do not abort engine creation.
func NewLuaEngine(config *LuaConfig) (*LuaEngine, error) {
	if config == nil {
		config = DefaultLuaConfig()
	}

	vm := lua.NewState()
	if vm == nil {
		return nil, fmt.Errorf("failed to create Lua state")
	}

	engine := &LuaEngine{
		vm:      vm,
		scripts: make(map[string]*LuaScript),
		timeout: config.Timeout,
		maxMem:  config.MaxMemory,
		sandbox: config.Sandbox,
	}

	// Setup Lua environment
	if err := engine.setupEnvironment(); err != nil {
		vm.Close()
		return nil, fmt.Errorf("failed to setup Lua environment: %w", err)
	}

	// Load scripts from directories
	for _, dir := range config.ScriptDirs {
		if err := engine.LoadScriptsFromDir(dir); err != nil {
			logger.Warn("Failed to load scripts from %s: %v", dir, err)
		}
	}

	return engine, nil
}

// setupEnvironment configures the Lua environment
func (e *LuaEngine) setupEnvironment() error {
	// Register GoCat API functions
	e.registerGoCatAPI()

	// Apply sandbox restrictions if enabled
	if e.sandbox {
		e.applySandbox()
	}

	return nil
}

// registerGoCatAPI registers GoCat-specific functions in Lua
func (e *LuaEngine) registerGoCatAPI() {
	// Network functions
	e.vm.SetGlobal("connect", e.vm.NewFunction(e.luaConnect))
	e.vm.SetGlobal("listen", e.vm.NewFunction(e.luaListen))
	e.vm.SetGlobal("send", e.vm.NewFunction(e.luaSend))
	e.vm.SetGlobal("receive", e.vm.NewFunction(e.luaReceive))
	e.vm.SetGlobal("close", e.vm.NewFunction(e.luaClose))

	// Utility functions
	e.vm.SetGlobal("log", e.vm.NewFunction(e.luaLog))
	e.vm.SetGlobal("sleep", e.vm.NewFunction(e.luaSleep))
	e.vm.SetGlobal("hex_encode", e.vm.NewFunction(e.luaHexEncode))
	e.vm.SetGlobal("hex_decode", e.vm.NewFunction(e.luaHexDecode))
	e.vm.SetGlobal("base64_encode", e.vm.NewFunction(e.luaBase64Encode))
	e.vm.SetGlobal("base64_decode", e.vm.NewFunction(e.luaBase64Decode))

	// File operations (if not sandboxed)
	if !e.sandbox {
		e.vm.SetGlobal("read_file", e.vm.NewFunction(e.luaReadFile))
		e.vm.SetGlobal("write_file", e.vm.NewFunction(e.luaWriteFile))
	}

	// Environment info
	envTable := e.vm.NewTable()
	envTable.RawSetString("version", lua.LString("GoCat 1.0"))
	envTable.RawSetString("platform", lua.LString(fmt.Sprintf("%s/%s", 
		os.Getenv("GOOS"), os.Getenv("GOARCH"))))
	e.vm.SetGlobal("gocat", envTable)
}

// applySandbox applies security restrictions
func (e *LuaEngine) applySandbox() {
	// Remove dangerous functions
	dangerousFuncs := []string{
		"os", "io", "package", "require", "dofile", "loadfile",
		"load", "loadstring", "debug", "collectgarbage",
	}

	for _, funcName := range dangerousFuncs {
		e.vm.SetGlobal(funcName, lua.LNil)
	}

	logger.Debug("Lua sandbox applied")
}

// LoadScript loads a Lua script from file
func (e *LuaEngine) LoadScript(path string) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read script file: %w", err)
	}

	name := filepath.Base(path)
	name = strings.TrimSuffix(name, filepath.Ext(name))

	// Compile script
	fn, err := e.vm.LoadString(string(content))
	if err != nil {
		return fmt.Errorf("failed to compile script: %w", err)
	}

	script := &LuaScript{
		Name:     name,
		Path:     path,
		Content:  string(content),
		Function: fn,
		Loaded:   time.Now(),
	}

	e.scripts[name] = script
	logger.Debug("Loaded Lua script: %s", name)

	return nil
}

// LoadScriptsFromDir loads all Lua scripts from a directory
func (e *LuaEngine) LoadScriptsFromDir(dir string) error {
	// Expand home directory
	if strings.HasPrefix(dir, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		dir = filepath.Join(home, dir[2:])
	}

	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil // Directory doesn't exist, skip silently
	}

	// Walk directory
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only load .lua files
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), ".lua") {
			if err := e.LoadScript(path); err != nil {
				logger.Warn("Failed to load script %s: %v", path, err)
			}
		}

		return nil
	})
}

// ExecuteScript executes a loaded script
func (e *LuaEngine) ExecuteScript(name string, args ...interface{}) ([]lua.LValue, error) {
	e.mutex.RLock()
	script, exists := e.scripts[name]
	e.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("script not found: %s", name)
	}

	return e.executeWithTimeout(script.Function, args...)
}

// ExecuteString executes Lua code from string
func (e *LuaEngine) ExecuteString(code string, args ...interface{}) ([]lua.LValue, error) {
	fn, err := e.vm.LoadString(code)
	if err != nil {
		return nil, fmt.Errorf("failed to compile code: %w", err)
	}

	return e.executeWithTimeout(fn, args...)
}

// executeWithTimeout executes a Lua function with timeout
func (e *LuaEngine) executeWithTimeout(fn *lua.LFunction, args ...interface{}) ([]lua.LValue, error) {
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	// Convert Go values to Lua values
	luaArgs := make([]lua.LValue, len(args))
	for i, arg := range args {
		switch v := arg.(type) {
		case string:
			luaArgs[i] = lua.LString(v)
		case int:
			luaArgs[i] = lua.LNumber(v)
		case float64:
			luaArgs[i] = lua.LNumber(v)
		case bool:
			luaArgs[i] = lua.LBool(v)
		default:
			// For complex types, store as user data
			ud := e.vm.NewUserData()
			ud.Value = v
			luaArgs[i] = ud
		}
	}

	// Execute in goroutine for timeout support
	resultChan := make(chan []lua.LValue, 1)
	errorChan := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				errorChan <- fmt.Errorf("script panic: %v", r)
			}
		}()

		// Push function and arguments
		e.vm.Push(fn)
		for _, arg := range luaArgs {
			e.vm.Push(arg)
		}

		// Call function
		err := e.vm.PCall(len(luaArgs), lua.MultRet, nil)
		if err != nil {
			errorChan <- err
			return
		}

		// Get results
		top := e.vm.GetTop()
		results := make([]lua.LValue, top)
		for i := 1; i <= top; i++ {
			results[i-1] = e.vm.Get(i)
		}
		e.vm.SetTop(0) // Clear stack

		resultChan <- results
	}()

	select {
	case results := <-resultChan:
		return results, nil
	case err := <-errorChan:
		return nil, err
	case <-ctx.Done():
		return nil, fmt.Errorf("script execution timeout")
	}
}

// GetLoadedScripts returns list of loaded scripts
func (e *LuaEngine) GetLoadedScripts() []string {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	scripts := make([]string, 0, len(e.scripts))
	for name := range e.scripts {
		scripts = append(scripts, name)
	}

	return scripts
}

// Close closes the Lua engine
func (e *LuaEngine) Close() {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if e.vm != nil {
		e.vm.Close()
		e.vm = nil
	}

	e.scripts = nil
	logger.Debug("Lua engine closed")
}

// Lua API function implementations

// luaConnect implements connect() function in Lua
func (e *LuaEngine) luaConnect(L *lua.LState) int {
	host := L.ToString(1)
	port := L.ToInt(2)
	protocol := L.ToString(3)

	if protocol == "" {
		protocol = "tcp"
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.Dial(protocol, addr)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	// Store connection in registry
	connID := fmt.Sprintf("conn_%p", conn)
	ud := L.NewUserData()
	ud.Value = conn
	L.SetGlobal(connID, ud)

	L.Push(lua.LString(connID))
	L.Push(lua.LNil)
	return 2
}

// luaListen implements listen() function in Lua
func (e *LuaEngine) luaListen(L *lua.LState) int {
	port := L.ToInt(1)
	protocol := L.ToString(2)

	if protocol == "" {
		protocol = "tcp"
	}

	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen(protocol, addr)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	// Store listener in registry
	listenerID := fmt.Sprintf("listener_%p", listener)
	ud := L.NewUserData()
	ud.Value = listener
	L.SetGlobal(listenerID, ud)

	L.Push(lua.LString(listenerID))
	L.Push(lua.LNil)
	return 2
}

// luaSend implements send() function in Lua
func (e *LuaEngine) luaSend(L *lua.LState) int {
	connID := L.ToString(1)
	data := L.ToString(2)

	connValue := L.GetGlobal(connID)
	if connValue == lua.LNil {
		L.Push(lua.LNumber(0))
		L.Push(lua.LString("connection not found"))
		return 2
	}

	// Extract connection from user data
	if ud, ok := connValue.(*lua.LUserData); ok {
		if conn, ok := ud.Value.(net.Conn); ok {
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
	}
	L.Push(lua.LNumber(0))
	L.Push(lua.LString("invalid connection"))
	return 2


}

// luaReceive implements receive() function in Lua
func (e *LuaEngine) luaReceive(L *lua.LState) int {
	connID := L.ToString(1)
	size := L.ToInt(2)

	if size <= 0 {
		size = 1024
	}

	connValue := L.GetGlobal(connID)
	if connValue == lua.LNil {
		L.Push(lua.LString(""))
		L.Push(lua.LString("connection not found"))
		return 2
	}

	// Extract connection from user data
	if ud, ok := connValue.(*lua.LUserData); ok {
		if conn, ok := ud.Value.(net.Conn); ok {
			buf := make([]byte, size)
			n, err := conn.Read(buf)
			if err != nil && err != io.EOF {
				L.Push(lua.LString(""))
				L.Push(lua.LString(err.Error()))
				return 2
			}
			L.Push(lua.LString(string(buf[:n])))
			L.Push(lua.LNil)
			return 2
		}
	}
	L.Push(lua.LString(""))
	L.Push(lua.LString("invalid connection"))
	return 2
}

// luaClose implements close() function in Lua
func (e *LuaEngine) luaClose(L *lua.LState) int {
	connID := L.ToString(1)

	connValue := L.GetGlobal(connID)
	if connValue == lua.LNil {
		L.Push(lua.LString("connection not found"))
		return 1
	}

	// Extract connection or listener from user data
	if ud, ok := connValue.(*lua.LUserData); ok {
		if conn, ok := ud.Value.(net.Conn); ok {
			err := conn.Close()
			L.SetGlobal(connID, lua.LNil) // Remove from registry
			if err != nil {
				L.Push(lua.LString(err.Error()))
				return 1
			}
		} else if listener, ok := ud.Value.(net.Listener); ok {
			err := listener.Close()
			L.SetGlobal(connID, lua.LNil) // Remove from registry
			if err != nil {
				L.Push(lua.LString(err.Error()))
				return 1
			}
		}
	}

	L.Push(lua.LNil)
	return 1
}

// luaLog implements log() function in Lua
func (e *LuaEngine) luaLog(L *lua.LState) int {
	level := L.ToString(1)
	message := L.ToString(2)

	switch strings.ToLower(level) {
	case "debug":
		logger.Debug("[Lua] %s", message)
	case "info":
		logger.Info("[Lua] %s", message)
	case "warn":
		logger.Warn("[Lua] %s", message)
	case "error":
		logger.Error("[Lua] %s", message)
	default:
		logger.Info("[Lua] %s", message)
	}

	return 0
}

// luaSleep implements sleep() function in Lua
func (e *LuaEngine) luaSleep(L *lua.LState) int {
	duration := L.ToNumber(1)
	time.Sleep(time.Duration(float64(duration) * float64(time.Second)))
	return 0
}

// luaHexEncode implements hex_encode() function in Lua
func (e *LuaEngine) luaHexEncode(L *lua.LState) int {
	data := L.ToString(1)
	encoded := fmt.Sprintf("%x", data)
	L.Push(lua.LString(encoded))
	return 1
}

// luaHexDecode implements hex_decode() function in Lua
func (e *LuaEngine) luaHexDecode(L *lua.LState) int {
	hexStr := L.ToString(1)
	var decoded []byte
	for i := 0; i < len(hexStr); i += 2 {
		if i+1 >= len(hexStr) {
			break
		}
		var b byte
		fmt.Sscanf(hexStr[i:i+2], "%02x", &b)
		decoded = append(decoded, b)
	}
	L.Push(lua.LString(string(decoded)))
	return 1
}

// luaBase64Encode implements base64_encode() function in Lua
func (e *LuaEngine) luaBase64Encode(L *lua.LState) int {
	data := L.ToString(1)
	encoded := base64.StdEncoding.EncodeToString([]byte(data))
	L.Push(lua.LString(encoded))
	return 1
}

// luaBase64Decode implements base64_decode() function in Lua
func (e *LuaEngine) luaBase64Decode(L *lua.LState) int {
	data := L.ToString(1)
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		L.Push(lua.LString(""))
		L.Push(lua.LString(err.Error()))
		return 2
	}
	L.Push(lua.LString(string(decoded)))
	L.Push(lua.LNil)
	return 2
}

// luaReadFile implements read_file() function in Lua (only if not sandboxed)
func (e *LuaEngine) luaReadFile(L *lua.LState) int {
	filename := L.ToString(1)
	content, err := os.ReadFile(filename)
	if err != nil {
		L.Push(lua.LString(""))
		L.Push(lua.LString(err.Error()))
		return 2
	}

	L.Push(lua.LString(string(content)))
	L.Push(lua.LNil)
	return 2
}

// luaWriteFile implements write_file() function in Lua (only if not sandboxed)
func (e *LuaEngine) luaWriteFile(L *lua.LState) int {
	filename := L.ToString(1)
	content := L.ToString(2)

	err := os.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		L.Push(lua.LString(err.Error()))
		return 1
	}

	L.Push(lua.LNil)
	return 1
}