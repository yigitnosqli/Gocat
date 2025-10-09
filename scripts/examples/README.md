# GoCat Lua Script Examples

This directory contains example Lua scripts demonstrating GoCat's scripting capabilities.

## Available Scripts

### 1. HTTP Client (`http_client.lua`)
Makes a simple HTTP GET request to a server.

```bash
gocat script scripts/examples/http_client.lua
```

**Features:**
- TCP connection
- HTTP request formatting
- Response parsing

### 2. Echo Server (`echo_server.lua`)
Demonstrates how to create a simple echo server.

```bash
gocat script scripts/examples/echo_server.lua
```

**Features:**
- Server socket creation
- Connection handling

### 3. Banner Grabber (`banner_grabber.lua`)
Connects to common services and grabs their banners.

```bash
gocat script scripts/examples/banner_grabber.lua
```

**Features:**
- Multiple service scanning
- Banner extraction
- Error handling

### 4. Port Scanner (`port_scanner.lua`)
Scans a range of ports on a target host.

```bash
gocat script scripts/examples/port_scanner.lua
```

**Features:**
- Port range scanning
- Connection testing
- Progress reporting

### 5. SSL Client (`ssl_client.lua`)
Demonstrates SSL/TLS connections.

```bash
gocat script scripts/examples/ssl_client.lua
```

**Features:**
- SSL/TLS connection
- HTTPS requests
- Certificate handling

### 6. Data Encoder (`data_encoder.lua`)
Shows encoding and decoding capabilities.

```bash
gocat script scripts/examples/data_encoder.lua
```

**Features:**
- Hex encoding/decoding
- Base64 encoding/decoding
- Data transformation

## Lua API Reference

### Network Functions

#### `connect(host, port, protocol)`
Connect to a remote host.
- **host**: Hostname or IP address
- **port**: Port number
- **protocol**: "tcp", "udp", "ssl", or "tls"
- **Returns**: connection object, error

#### `listen(port, protocol)`
Listen for incoming connections.
- **port**: Port number to listen on
- **protocol**: "tcp" or "udp"
- **Returns**: listener object, error

#### `send(conn, data)`
Send data through a connection.
- **conn**: Connection object
- **data**: Data to send (string)
- **Returns**: bytes sent, error

#### `receive(conn, size)`
Receive data from a connection.
- **conn**: Connection object
- **size**: Maximum bytes to receive
- **Returns**: received data, error

#### `close(conn)`
Close a connection.
- **conn**: Connection object
- **Returns**: success (boolean)

### Utility Functions

#### `log(level, message)`
Log a message.
- **level**: "debug", "info", "warn", or "error"
- **message**: Message to log

#### `sleep(seconds)`
Sleep for specified duration.
- **seconds**: Duration in seconds (can be fractional)

#### `hex_encode(data)`
Encode data as hexadecimal.
- **data**: Data to encode
- **Returns**: hex string

#### `hex_decode(hex)`
Decode hexadecimal string.
- **hex**: Hex string to decode
- **Returns**: decoded data

#### `base64_encode(data)`
Encode data as base64.
- **data**: Data to encode
- **Returns**: base64 string

#### `base64_decode(b64)`
Decode base64 string.
- **b64**: Base64 string to decode
- **Returns**: decoded data

### Global Variables

#### `gocat.version`
GoCat version string

#### `gocat.platform`
Platform information (OS/architecture)

## Creating Custom Scripts

Here's a template for creating your own scripts:

```lua
-- My Custom Script
-- Description of what it does

-- Configuration
local config = {
    host = "example.com",
    port = 80,
    timeout = 10
}

-- Main function
local function main()
    log("info", "Starting custom script...")
    
    -- Your code here
    local conn, err = connect(config.host, config.port, "tcp")
    if err then
        log("error", "Connection failed: " .. err)
        return
    end
    
    log("info", "Connected successfully")
    
    -- Do something with the connection
    send(conn, "Hello, World!\n")
    local response, recv_err = receive(conn, 1024)
    
    if response then
        log("info", "Received: " .. response)
    end
    
    close(conn)
    log("info", "Script completed")
end

-- Run main function
main()
```

## Best Practices

1. **Error Handling**: Always check for errors from network functions
2. **Resource Cleanup**: Close connections when done
3. **Logging**: Use appropriate log levels
4. **Timeouts**: Implement timeouts for long-running operations
5. **Rate Limiting**: Add delays between operations to avoid overwhelming targets

## Security Considerations

- Scripts run with the same privileges as GoCat
- Be careful when connecting to untrusted hosts
- Validate all input data
- Use SSL/TLS for sensitive communications
- Follow responsible disclosure practices

## Contributing

To contribute new example scripts:

1. Create a well-documented script
2. Add it to this directory
3. Update this README
4. Test thoroughly
5. Submit a pull request

## License

These examples are part of the GoCat project and are licensed under the MIT License.
