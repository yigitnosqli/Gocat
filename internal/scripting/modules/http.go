package modules

import (
	"io"
	"net/http"
	"strings"
	"time"

	lua "github.com/yuin/gopher-lua"
)

// RegisterHTTPModule registers HTTP-related Lua functions
func RegisterHTTPModule(L *lua.LState) {
	httpModule := L.NewTable()

	L.SetField(httpModule, "get", L.NewFunction(luaHTTPGet))
	L.SetField(httpModule, "post", L.NewFunction(luaHTTPPost))
	L.SetField(httpModule, "put", L.NewFunction(luaHTTPPut))
	L.SetField(httpModule, "delete", L.NewFunction(luaHTTPDelete))
	L.SetField(httpModule, "request", L.NewFunction(luaHTTPRequest))
	L.SetField(httpModule, "download", L.NewFunction(luaHTTPDownload))

	L.SetGlobal("http", httpModule)
}

// luaHTTPGet implements http.get(url)
func luaHTTPGet(L *lua.LState) int {
	url := L.ToString(1)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	result := L.NewTable()
	result.RawSetString("status", lua.LNumber(resp.StatusCode))
	result.RawSetString("body", lua.LString(string(body)))

	headers := L.NewTable()
	for key, values := range resp.Header {
		headers.RawSetString(key, lua.LString(strings.Join(values, ", ")))
	}
	result.RawSetString("headers", headers)

	L.Push(result)
	L.Push(lua.LNil)
	return 2
}

// luaHTTPPost implements http.post(url, data)
func luaHTTPPost(L *lua.LState) int {
	url := L.ToString(1)
	data := L.ToString(2)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(url, "application/x-www-form-urlencoded", strings.NewReader(data))
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	result := L.NewTable()
	result.RawSetString("status", lua.LNumber(resp.StatusCode))
	result.RawSetString("body", lua.LString(string(body)))

	L.Push(result)
	L.Push(lua.LNil)
	return 2
}

// luaHTTPPut implements http.put(url, data)
func luaHTTPPut(L *lua.LState) int {
	url := L.ToString(1)
	data := L.ToString(2)

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodPut, url, strings.NewReader(data))
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	resp, err := client.Do(req)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	result := L.NewTable()
	result.RawSetString("status", lua.LNumber(resp.StatusCode))
	result.RawSetString("body", lua.LString(string(body)))

	L.Push(result)
	L.Push(lua.LNil)
	return 2
}

// luaHTTPDelete implements http.delete(url)
func luaHTTPDelete(L *lua.LState) int {
	url := L.ToString(1)

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	resp, err := client.Do(req)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	result := L.NewTable()
	result.RawSetString("status", lua.LNumber(resp.StatusCode))
	result.RawSetString("body", lua.LString(string(body)))

	L.Push(result)
	L.Push(lua.LNil)
	return 2
}

// luaHTTPRequest implements http.request(method, url, headers, body)
func luaHTTPRequest(L *lua.LState) int {
	method := L.ToString(1)
	url := L.ToString(2)
	headersTable := L.ToTable(3)
	body := L.ToString(4)

	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	// Set headers
	if headersTable != nil {
		headersTable.ForEach(func(key, value lua.LValue) {
			req.Header.Set(key.String(), value.String())
		})
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	result := L.NewTable()
	result.RawSetString("status", lua.LNumber(resp.StatusCode))
	result.RawSetString("body", lua.LString(string(respBody)))

	headers := L.NewTable()
	for key, values := range resp.Header {
		headers.RawSetString(key, lua.LString(strings.Join(values, ", ")))
	}
	result.RawSetString("headers", headers)

	L.Push(result)
	L.Push(lua.LNil)
	return 2
}

// luaHTTPDownload implements http.download(url, filepath)
func luaHTTPDownload(L *lua.LState) int {
	url := L.ToString(1)
	_ = L.ToString(2) // filepath - currently unused

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString(err.Error()))
		return 2
	}
	defer resp.Body.Close()

	// TODO: Implement file writing

	L.Push(lua.LBool(true))
	L.Push(lua.LNil)
	return 2
}
