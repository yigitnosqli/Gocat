package scripting

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ibrahmsql/gocat/internal/logger"
	"github.com/ibrahmsql/gocat/internal/scripting/modules"
	lua "github.com/yuin/gopher-lua"
)

// Engine represents a Lua scripting engine for GoCat
type Engine struct {
	L             *lua.LState
	config        *Config
	ctx           context.Context
	cancel        context.CancelFunc
	loadedScripts map[string]bool
}

// Config holds configuration for the Lua engine
type Config struct {
	MaxExecutionTime time.Duration
	MaxMemory        int64
	RestrictedMode   bool
	AllowedHosts     []string
	DeniedHosts      []string
	ModulesPath      string
	Debug            bool
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		MaxExecutionTime: 30 * time.Second,
		MaxMemory:        64 * 1024 * 1024, // 64MB
		RestrictedMode:   false,
		Debug:            false,
	}
}

// NewEngine creates a new Lua engine with configuration
func NewEngine(config *Config) *Engine {
	if config == nil {
		config = DefaultConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())
	L := lua.NewState()
	
	// Configure Lua state limits
	L.SetMx(int(config.MaxMemory))
	
	engine := &Engine{
		L:             L,
		config:        config,
		ctx:           ctx,
		cancel:        cancel,
		loadedScripts: make(map[string]bool),
	}

	// Register all modules
	engine.registerModules()

	return engine
}

// registerModules registers all Lua modules
func (e *Engine) registerModules() {
	// Core modules
	modules.RegisterNetworkModule(e.L, e.config.RestrictedMode)
	modules.RegisterHTTPModule(e.L)
	modules.RegisterCryptoModule(e.L)
	modules.RegisterSystemModule(e.L, e.config.RestrictedMode)
	modules.RegisterFileModule(e.L)
	modules.RegisterTimeModule(e.L)
	modules.RegisterUIModule(e.L)
	modules.RegisterJSONModule(e.L)
	
	// Utility functions (backward compatibility)
	e.registerUtilityFunctions()
	
	// GoCat environment info
	e.registerGoCatInfo()
	
	if e.config.Debug {
		logger.Debug("Registered all Lua modules")
	}
}

// registerUtilityFunctions registers utility functions for backward compatibility
func (e *Engine) registerUtilityFunctions() {
	// Legacy functions that are directly in global scope
	e.L.SetGlobal("log", e.L.NewFunction(modules.LuaLog))
	e.L.SetGlobal("sleep", e.L.NewFunction(modules.LuaSleep))
	e.L.SetGlobal("print", e.L.NewFunction(modules.LuaPrint))
}

// registerGoCatInfo registers GoCat environment information
func (e *Engine) registerGoCatInfo() {
	gocatTable := e.L.NewTable()
	gocatTable.RawSetString("version", lua.LString("1.0.0"))
	gocatTable.RawSetString("platform", lua.LString("cross-platform"))
	
	// Add configuration info
	configTable := e.L.NewTable()
	configTable.RawSetString("restricted", lua.LBool(e.config.RestrictedMode))
	configTable.RawSetString("maxMemory", lua.LNumber(e.config.MaxMemory))
	configTable.RawSetString("maxExecutionTime", lua.LNumber(e.config.MaxExecutionTime.Seconds()))
	gocatTable.RawSetString("config", configTable)
	
	e.L.SetGlobal("gocat", gocatTable)
}

// LoadScript loads a Lua script from file
func (e *Engine) LoadScript(scriptPath string) error {
	if e.L == nil {
		return fmt.Errorf("lua engine is closed")
	}

	// Check if script already loaded
	if e.loadedScripts[scriptPath] {
		if e.config.Debug {
			logger.Debug("Script already loaded: %s", scriptPath)
		}
		return nil
	}

	// Check file size
	info, err := os.Stat(scriptPath)
	if err != nil {
		return fmt.Errorf("failed to stat script: %w", err)
	}
	
	if info.Size() > 10*1024*1024 { // 10MB limit
		return fmt.Errorf("script too large: %d bytes", info.Size())
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
	
	if e.config.Debug {
		logger.Debug("Successfully loaded script: %s", scriptPath)
	}
	
	return nil
}

// LoadString loads and executes Lua code from a string
func (e *Engine) LoadString(code string) error {
	if e.L == nil {
		return fmt.Errorf("lua engine is closed")
	}

	if err := e.L.DoString(code); err != nil {
		return fmt.Errorf("failed to execute code: %w", err)
	}

	return nil
}

// ExecuteFunction executes a specific function in the loaded script
func (e *Engine) ExecuteFunction(functionName string, args ...lua.LValue) ([]lua.LValue, error) {
	if e.L == nil {
		return nil, fmt.Errorf("lua engine is closed")
	}

	// Get the function
	fn := e.L.GetGlobal(functionName)
	if fn.Type() != lua.LTFunction {
		return nil, fmt.Errorf("function '%s' not found or not a function", functionName)
	}

	// Set execution timeout
	done := make(chan bool)
	var execErr error
	var results []lua.LValue

	go func() {
		// Call function with arguments
		err := e.L.CallByParam(lua.P{
			Fn:      fn,
			NRet:    lua.MultRet,
			Protect: true,
		}, args...)

		if err != nil {
			execErr = fmt.Errorf("error executing function '%s': %w", functionName, err)
			done <- true
			return
		}

		// Collect return values
		top := e.L.GetTop()
		for i := 1; i <= top; i++ {
			results = append(results, e.L.Get(i))
		}
		e.L.SetTop(0) // Clear stack
		
		done <- true
	}()

	// Wait for completion or timeout
	select {
	case <-done:
		return results, execErr
	case <-time.After(e.config.MaxExecutionTime):
		return nil, fmt.Errorf("function execution timeout after %v", e.config.MaxExecutionTime)
	case <-e.ctx.Done():
		return nil, fmt.Errorf("engine context cancelled")
	}
}

// GetGlobal gets a global variable from Lua state
func (e *Engine) GetGlobal(name string) lua.LValue {
	if e.L == nil {
		return lua.LNil
	}
	return e.L.GetGlobal(name)
}

// SetGlobal sets a global variable in Lua state
func (e *Engine) SetGlobal(name string, value lua.LValue) {
	if e.L != nil {
		e.L.SetGlobal(name, value)
	}
}

// Close closes the Lua engine and releases resources
func (e *Engine) Close() {
	if e.cancel != nil {
		e.cancel()
	}
	
	if e.L != nil {
		e.L.Close()
		e.L = nil
	}
	
	e.loadedScripts = nil
}

// IsRestricted returns whether the engine is in restricted mode
func (e *Engine) IsRestricted() bool {
	return e.config.RestrictedMode
}

// SetDebug enables or disables debug mode
func (e *Engine) SetDebug(debug bool) {
	e.config.Debug = debug
}
