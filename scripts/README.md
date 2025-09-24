# GoCat Lua Scripts

This directory contains Lua scripts designed for **GoCat**. These scripts use GoCat’s Lua engine feature to automate various network tasks.

## Available Scripts

### 1. `port_scanner.lua`

**Purpose:** Performs port scanning on target hosts.

**Features:**

* Scans specified port ranges
* Detects and logs open ports
* Displays progress indicators
* Rate limiting to prevent overwhelming targets

**Usage:**

```lua
-- Edit parameters within the script:
local host = "127.0.0.1"
local start_port = 20
local end_port = 100
```

### 2. `banner_grabber.lua`

**Purpose:** Collects banner information from services.

**Features:**

* Retrieves banners of common services (FTP, SSH, HTTP, etc.)
* Can send HTTP requests
* Useful for service fingerprinting

**Usage:**

```lua
-- Change the target host:
local target_host = "127.0.0.1"
```

### 3. `http_client.lua`

**Purpose:** Simple HTTP client implementation.

**Features:**

* Supports GET and POST requests
* Custom header support
* Response parsing
* HTTP/1.1 compliant

**Usage:**

```lua
local response = simple_get("example.com", 80, "/")
local post_response = simple_post("api.example.com", 80, "/data", "key=value")
```

### 4. `chat_bot.lua`

**Purpose:** Simple chat bot service.

**Features:**

* Command-based responses
* Echo command
* Extendable response system
* Chat session management

**Usage:**

```lua
-- For testing:
test_chat_responses()

-- To start the server:
start_chat_server(8888)
```

### 5. `network_monitor.lua`

**Purpose:** Monitors network connections.

**Features:**

* Multi-target monitoring
* Uptime calculation
* Failure alerting
* Ping test function

**Usage:**

```lua
-- Ping test:
ping_test("8.8.8.8", 4)

-- Continuous monitoring:
monitor_targets()
```

### 6. `data_encoder.lua`

**Purpose:** Data encoding/decoding utilities.

**Features:**

* Hex, Base64, and URL encoding
* HTML entity encoding
* Caesar cipher (ROT13)
* Morse code encoding
* Binary encoding

**Usage:**

```lua
local encoded = hex_encode("Hello World")
local decoded = hex_decode(encoded)
local rot13 = caesar_cipher("Hello", 13)
```

## GoCat Lua API

These scripts make use of the following GoCat Lua API functions:

### Network Functions

* `connect(host, port, protocol)` – Establish a connection
* `listen(port, protocol)` – Start listening
* `send(conn, data)` – Send data
* `receive(conn, size)` – Receive data
* `close(conn)` – Close a connection

### Utility Functions

* `log(level, message)` – Log a message
* `sleep(seconds)` – Pause execution
* `hex_encode(data)` – Hex encoding
* `hex_decode(hex)` – Hex decoding
* `base64_encode(data)` – Base64 encoding
* `base64_decode(b64)` – Base64 decoding

### Environment Info

* `gocat.version` – GoCat version
* `gocat.platform` – Platform information

## Running Scripts

To run Lua scripts in GoCat:

1. **Through the TUI:**

   * Navigate to the script menu
   * Select the desired script
   * Execute it

2. **From the command line:**

   ```bash
   gocat script run port_scanner.lua
   ```

3. **Programmatically:**

   ```go
   engine := scripting.NewLuaEngine(nil)
   engine.LoadScript("scripts/port_scanner.lua")
   engine.ExecuteScript("port_scanner")
   ```

## Script Development Guidelines

When creating new scripts:

1. Use **error handling**
2. Add **logging**
3. Apply **rate limiting**
4. Include **configurable parameters**
5. Provide **documentation**

## Security

* Scripts run in a sandboxed environment
* File system access is restricted
* Network access is controlled
* Dangerous functions are disabled

## Examples

### Simple Port Scan

```lua
local open_ports = scan_range("192.168.1.1", 80, 443)
for _, port in ipairs(open_ports) do
    log("info", "Open port: " .. port)
end
```

### HTTP Health Check

```lua
local response = simple_get("api.example.com", 80, "/health")
if response and response.status_code == 200 then
    log("info", "Service is healthy")
else
    log("error", "Service is down")
end
```

### Data Encoding

```lua
local secret = "my secret data"
local encoded = base64_encode(secret)
local decoded = base64_decode(encoded)
log("info", "Original: " .. secret)
log("info", "Encoded: " .. encoded)
log("info", "Decoded: " .. decoded)
```

These scripts demonstrate GoCat’s powerful Lua scripting capabilities and help automate a wide range of network tasks.
