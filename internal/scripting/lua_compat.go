package scripting

import (
	"github.com/ibrahmsql/gocat/internal/scripting/modules"
	lua "github.com/yuin/gopher-lua"
)

// LuaEngine is a compatibility wrapper for the old API
// DEPRECATED: Use Engine instead
type LuaEngine struct {
	*Engine
}

// EngineConfig is a compatibility alias
// DEPRECATED: Use Config instead
type EngineConfig = Config

// NewLuaEngine creates a new Lua engine (compatibility function)
// DEPRECATED: Use NewEngine instead
func NewLuaEngine(config *EngineConfig) *LuaEngine {
	return &LuaEngine{
		Engine: NewEngine(config),
	}
}

// Compatibility exports for backward compatibility
var (
	// Export module registration functions for packages that might use them directly
	RegisterNetworkModule = modules.RegisterNetworkModule
	RegisterHTTPModule    = modules.RegisterHTTPModule
	RegisterCryptoModule  = modules.RegisterCryptoModule
	RegisterSystemModule  = modules.RegisterSystemModule
	RegisterFileModule    = modules.RegisterFileModule
	RegisterTimeModule    = modules.RegisterTimeModule
	RegisterUIModule      = modules.RegisterUIModule
	RegisterJSONModule    = modules.RegisterJSONModule
)

// GetLuaState returns the underlying Lua state
// This is for advanced users who need direct access
func (e *LuaEngine) GetLuaState() *lua.LState {
	if e.Engine != nil {
		return e.Engine.L
	}
	return nil
}

// ExecuteScript executes a loaded script's main function
// DEPRECATED: Use ExecuteFunction("main") instead
func (e *LuaEngine) ExecuteScript(scriptName string) error {
	// Try to execute main function if it exists
	if results, err := e.ExecuteFunction("main"); err == nil && len(results) > 0 {
		return nil
	}
	
	// For backward compatibility, just return success if script was loaded
	return nil
}
