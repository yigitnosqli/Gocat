package modules

import (
	"time"

	lua "github.com/yuin/gopher-lua"
)

// RegisterTimeModule registers time-related Lua functions
func RegisterTimeModule(L *lua.LState) {
	timeModule := L.NewTable()
	
	L.SetField(timeModule, "now", L.NewFunction(luaTimeNow))
	L.SetField(timeModule, "sleep", L.NewFunction(LuaSleep))
	L.SetField(timeModule, "format", L.NewFunction(luaTimeFormat))
	L.SetField(timeModule, "parse", L.NewFunction(luaTimeParse))
	L.SetField(timeModule, "unix", L.NewFunction(luaTimeUnix))
	L.SetField(timeModule, "since", L.NewFunction(luaTimeSince))
	
	L.SetGlobal("time", timeModule)
}

// luaTimeNow implements time.now()
func luaTimeNow(L *lua.LState) int {
	L.Push(lua.LNumber(time.Now().Unix()))
	return 1
}

// LuaSleep implements time.sleep(seconds) - exported for backward compatibility
func LuaSleep(L *lua.LState) int {
	seconds := L.ToNumber(1)
	if seconds > 0 {
		time.Sleep(time.Duration(float64(seconds) * float64(time.Second)))
	}
	return 0
}

// luaTimeFormat implements time.format(timestamp, format)
func luaTimeFormat(L *lua.LState) int {
	timestamp := L.ToInt64(1)
	format := L.ToString(2)
	
	if format == "" {
		format = "2006-01-02 15:04:05"
	}
	
	t := time.Unix(timestamp, 0)
	L.Push(lua.LString(t.Format(format)))
	return 1
}

// luaTimeParse implements time.parse(timestr, format)
func luaTimeParse(L *lua.LState) int {
	timestr := L.ToString(1)
	format := L.ToString(2)
	
	if format == "" {
		format = "2006-01-02 15:04:05"
	}
	
	t, err := time.Parse(format, timestr)
	if err != nil {
		L.Push(lua.LNumber(0))
		L.Push(lua.LString(err.Error()))
		return 2
	}
	
	L.Push(lua.LNumber(t.Unix()))
	L.Push(lua.LNil)
	return 2
}

// luaTimeUnix implements time.unix(timestamp)
func luaTimeUnix(L *lua.LState) int {
	timestamp := L.ToInt64(1)
	t := time.Unix(timestamp, 0)
	
	result := L.NewTable()
	result.RawSetString("year", lua.LNumber(t.Year()))
	result.RawSetString("month", lua.LNumber(t.Month()))
	result.RawSetString("day", lua.LNumber(t.Day()))
	result.RawSetString("hour", lua.LNumber(t.Hour()))
	result.RawSetString("minute", lua.LNumber(t.Minute()))
	result.RawSetString("second", lua.LNumber(t.Second()))
	result.RawSetString("weekday", lua.LString(t.Weekday().String()))
	
	L.Push(result)
	return 1
}

// luaTimeSince implements time.since(timestamp)
func luaTimeSince(L *lua.LState) int {
	timestamp := L.ToInt64(1)
	t := time.Unix(timestamp, 0)
	duration := time.Since(t)
	
	L.Push(lua.LNumber(duration.Seconds()))
	return 1
}
