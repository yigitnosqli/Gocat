package modules

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/ibrahmsql/gocat/internal/logger"
	lua "github.com/yuin/gopher-lua"
)

// RegisterUIModule registers UI/output-related Lua functions
func RegisterUIModule(L *lua.LState) {
	uiModule := L.NewTable()
	
	// Output functions
	L.SetField(uiModule, "print", L.NewFunction(LuaPrint))
	L.SetField(uiModule, "printf", L.NewFunction(luaPrintf))
	L.SetField(uiModule, "error", L.NewFunction(luaError))
	L.SetField(uiModule, "warn", L.NewFunction(luaWarn))
	L.SetField(uiModule, "info", L.NewFunction(luaInfo))
	L.SetField(uiModule, "success", L.NewFunction(luaSuccess))
	L.SetField(uiModule, "debug", L.NewFunction(luaDebug))
	
	// Color functions
	L.SetField(uiModule, "red", L.NewFunction(luaRed))
	L.SetField(uiModule, "green", L.NewFunction(luaGreen))
	L.SetField(uiModule, "yellow", L.NewFunction(luaYellow))
	L.SetField(uiModule, "blue", L.NewFunction(luaBlue))
	L.SetField(uiModule, "cyan", L.NewFunction(luaCyan))
	L.SetField(uiModule, "magenta", L.NewFunction(luaMagenta))
	L.SetField(uiModule, "white", L.NewFunction(luaWhite))
	
	// Progress indicators
	L.SetField(uiModule, "progress", L.NewFunction(luaProgress))
	L.SetField(uiModule, "clear", L.NewFunction(luaClear))
	
	L.SetGlobal("ui", uiModule)
}

// LuaPrint implements ui.print(message) - exported for backward compatibility
func LuaPrint(L *lua.LState) int {
	message := L.ToString(1)
	fmt.Println(message)
	return 0
}

// luaPrintf implements ui.printf(format, ...)
func luaPrintf(L *lua.LState) int {
	format := L.ToString(1)
	
	args := []interface{}{}
	for i := 2; i <= L.GetTop(); i++ {
		v := L.Get(i)
		switch v.Type() {
		case lua.LTString:
			args = append(args, v.String())
		case lua.LTNumber:
			args = append(args, float64(v.(lua.LNumber)))
		case lua.LTBool:
			args = append(args, bool(v.(lua.LBool)))
		default:
			args = append(args, v.String())
		}
	}
	
	fmt.Printf(format+"\n", args...)
	return 0
}

// luaError implements ui.error(message)
func luaError(L *lua.LState) int {
	message := L.ToString(1)
	color.Red("❌ ERROR: %s", message)
	return 0
}

// luaWarn implements ui.warn(message)
func luaWarn(L *lua.LState) int {
	message := L.ToString(1)
	color.Yellow("⚠️  WARN: %s", message)
	return 0
}

// luaInfo implements ui.info(message)
func luaInfo(L *lua.LState) int {
	message := L.ToString(1)
	color.Cyan("ℹ️  INFO: %s", message)
	return 0
}

// luaSuccess implements ui.success(message)
func luaSuccess(L *lua.LState) int {
	message := L.ToString(1)
	color.Green("✅ SUCCESS: %s", message)
	return 0
}

// luaDebug implements ui.debug(message)
func luaDebug(L *lua.LState) int {
	message := L.ToString(1)
	logger.Debug("[Script] %s", message)
	return 0
}

// Color output functions

// luaRed implements ui.red(message)
func luaRed(L *lua.LState) int {
	message := L.ToString(1)
	color.Red(message)
	return 0
}

// luaGreen implements ui.green(message)
func luaGreen(L *lua.LState) int {
	message := L.ToString(1)
	color.Green(message)
	return 0
}

// luaYellow implements ui.yellow(message)
func luaYellow(L *lua.LState) int {
	message := L.ToString(1)
	color.Yellow(message)
	return 0
}

// luaBlue implements ui.blue(message)
func luaBlue(L *lua.LState) int {
	message := L.ToString(1)
	color.Blue(message)
	return 0
}

// luaCyan implements ui.cyan(message)
func luaCyan(L *lua.LState) int {
	message := L.ToString(1)
	color.Cyan(message)
	return 0
}

// luaMagenta implements ui.magenta(message)
func luaMagenta(L *lua.LState) int {
	message := L.ToString(1)
	color.Magenta(message)
	return 0
}

// luaWhite implements ui.white(message)
func luaWhite(L *lua.LState) int {
	message := L.ToString(1)
	color.White(message)
	return 0
}

// luaProgress implements ui.progress(current, total, message)
func luaProgress(L *lua.LState) int {
	current := L.ToInt(1)
	total := L.ToInt(2)
	message := L.ToString(3)
	
	if total > 0 {
		percent := (current * 100) / total
		bar := makeProgressBar(percent, 30)
		fmt.Printf("\r%s %s %d%% (%d/%d)", bar, message, percent, current, total)
		
		if current >= total {
			fmt.Println() // New line when complete
		}
	}
	
	return 0
}

// luaClear implements ui.clear()
func luaClear(L *lua.LState) int {
	fmt.Print("\033[H\033[2J") // ANSI escape codes to clear screen
	return 0
}

// LuaLog implements log(message, level) - exported for backward compatibility
func LuaLog(L *lua.LState) int {
	message := L.ToString(1)
	level := L.ToString(2)
	
	switch level {
	case "error":
		logger.Error("[Script] %s", message)
	case "warn":
		logger.Warn("[Script] %s", message)
	case "debug":
		logger.Debug("[Script] %s", message)
	default:
		logger.Info("[Script] %s", message)
	}
	
	return 0
}

// makeProgressBar creates a visual progress bar
func makeProgressBar(percent, width int) string {
	if percent > 100 {
		percent = 100
	}
	if percent < 0 {
		percent = 0
	}
	
	filled := (percent * width) / 100
	empty := width - filled
	
	bar := "["
	for i := 0; i < filled; i++ {
		bar += "█"
	}
	for i := 0; i < empty; i++ {
		bar += "░"
	}
	bar += "]"
	
	return bar
}
