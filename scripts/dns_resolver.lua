-- DNS Resolver Script for GoCat
-- Performs DNS queries and analysis

local dns_record_types = {
    A = 1,      -- IPv4 address
    NS = 2,     -- Name server
    CNAME = 5,  -- Canonical name
    SOA = 6,    -- Start of authority
    PTR = 12,   -- Pointer record
    MX = 15,    -- Mail exchange
    TXT = 16,   -- Text record
    AAAA = 28,  -- IPv6 address
    SRV = 33    -- Service record
}

local dns_servers = {
    "8.8.8.8",      -- Google DNS
    "8.8.4.4",      -- Google DNS Secondary
    "1.1.1.1",      -- Cloudflare DNS
    "1.0.0.1",      -- Cloudflare DNS Secondary
    "208.67.222.222", -- OpenDNS
    "208.67.220.220"  -- OpenDNS Secondary
}

function create_dns_query(domain, record_type)
    record_type = record_type or dns_record_types.A
    
    -- DNS header (12 bytes)
    local transaction_id = math.random(1, 65535)
    local flags = 0x0100 -- Standard query with recursion desired
    local questions = 1
    local answers = 0
    local authority = 0
    local additional = 0
    
    local header = string.char(
        math.floor(transaction_id / 256), transaction_id % 256,  -- Transaction ID
        math.floor(flags / 256), flags % 256,                   -- Flags
        math.floor(questions / 256), questions % 256,           -- Questions
        math.floor(answers / 256), answers % 256,               -- Answers
        math.floor(authority / 256), authority % 256,           -- Authority
        math.floor(additional / 256), additional % 256          -- Additional
    )
    
    -- Question section
    local question = ""
    
    -- Encode domain name
    for part in string.gmatch(domain, "[^%.]+") do
        question = question .. string.char(#part) .. part
    end
    question = question .. string.char(0) -- End of domain name
    
    -- Query type and class
    question = question .. string.char(
        math.floor(record_type / 256), record_type % 256,  -- Type
        0, 1  -- Class (IN = Internet)
    )
    
    return header .. question, transaction_id
end

function parse_dns_response(response, transaction_id)
    if not response or #response < 12 then
        return nil, "Invalid DNS response"
    end
    
    -- Parse header
    local resp_id = string.byte(response, 1) * 256 + string.byte(response, 2)
    if resp_id ~= transaction_id then
        return nil, "Transaction ID mismatch"
    end
    
    local flags = string.byte(response, 3) * 256 + string.byte(response, 4)
    local rcode = flags & 0x0F
    
    if rcode ~= 0 then
        local error_codes = {
            [1] = "Format error",
            [2] = "Server failure",
            [3] = "Name error (domain does not exist)",
            [4] = "Not implemented",
            [5] = "Refused"
        }
        return nil, error_codes[rcode] or ("DNS error code: " .. rcode)
    end
    
    local questions = string.byte(response, 5) * 256 + string.byte(response, 6)
    local answers = string.byte(response, 7) * 256 + string.byte(response, 8)
    
    log("debug", "DNS Response: " .. answers .. " answers for " .. questions .. " questions")
    
    -- Skip question section (we know what we asked)
    local offset = 13 -- Start after header
    
    -- Skip question section
    for i = 1, questions do
        -- Skip domain name
        while offset <= #response do
            local len = string.byte(response, offset)
            if len == 0 then
                offset = offset + 1
                break
            elseif len >= 192 then -- Compression pointer
                offset = offset + 2
                break
            else
                offset = offset + len + 1
            end
        end
        offset = offset + 4 -- Skip type and class
    end
    
    -- Parse answer section
    local results = {}
    for i = 1, answers do
        if offset + 10 > #response then
            break
        end
        
        -- Skip name (assume compression pointer)
        offset = offset + 2
        
        local record_type = string.byte(response, offset) * 256 + string.byte(response, offset + 1)
        local record_class = string.byte(response, offset + 2) * 256 + string.byte(response, offset + 3)
        local ttl = string.byte(response, offset + 4) * 16777216 + 
                   string.byte(response, offset + 5) * 65536 + 
                   string.byte(response, offset + 6) * 256 + 
                   string.byte(response, offset + 7)
        local data_length = string.byte(response, offset + 8) * 256 + string.byte(response, offset + 9)
        
        offset = offset + 10
        
        local record_data = ""
        if record_type == dns_record_types.A and data_length == 4 then
            -- IPv4 address
            record_data = string.format("%d.%d.%d.%d",
                string.byte(response, offset),
                string.byte(response, offset + 1),
                string.byte(response, offset + 2),
                string.byte(response, offset + 3)
            )
        elseif record_type == dns_record_types.AAAA and data_length == 16 then
            -- IPv6 address (simplified)
            record_data = "IPv6 address (parsing not implemented)"
        else
            -- Other record types
            record_data = "Data length: " .. data_length .. " bytes"
        end
        
        table.insert(results, {
            type = record_type,
            class = record_class,
            ttl = ttl,
            data = record_data
        })
        
        offset = offset + data_length
    end
    
    return results, nil
end

function dns_query(domain, record_type, dns_server, timeout)
    record_type = record_type or dns_record_types.A
    dns_server = dns_server or "8.8.8.8"
    timeout = timeout or 5
    
    log("info", "Querying " .. domain .. " (type " .. record_type .. ") via " .. dns_server)
    
    local query, transaction_id = create_dns_query(domain, record_type)
    
    local conn, err = connect(dns_server, 53, "udp")
    if not conn then
        return nil, "Failed to connect to DNS server: " .. (err or "unknown error")
    end
    
    -- Send DNS query
    local sent, send_err = send(conn, query)
    if not sent or sent == 0 then
        close(conn)
        return nil, "Failed to send DNS query: " .. (send_err or "unknown error")
    end
    
    -- Receive response
    sleep(1) -- Wait for response
    local response, recv_err = receive(conn, 512)
    close(conn)
    
    if not response or #response == 0 then
        return nil, "No DNS response received: " .. (recv_err or "timeout")
    end
    
    return parse_dns_response(response, transaction_id)
end

function resolve_domain(domain, record_types)
    record_types = record_types or {dns_record_types.A}
    
    log("info", "Resolving domain: " .. domain)
    
    local results = {}
    
    for _, record_type in ipairs(record_types) do
        local records, err = dns_query(domain, record_type)
        
        if records then
            for _, record in ipairs(records) do
                table.insert(results, {
                    domain = domain,
                    type = record_type,
                    ttl = record.ttl,
                    data = record.data
                })
                
                local type_name = "UNKNOWN"
                for name, value in pairs(dns_record_types) do
                    if value == record_type then
                        type_name = name
                        break
                    end
                end
                
                log("info", domain .. " " .. type_name .. " " .. record.data .. " (TTL: " .. record.ttl .. ")")
            end
        else
            log("warn", "Failed to resolve " .. domain .. " (type " .. record_type .. "): " .. (err or "unknown error"))
        end
        
        sleep(0.5) -- Rate limiting
    end
    
    return results
end

function dns_server_test(domain, servers)
    domain = domain or "google.com"
    servers = servers or dns_servers
    
    log("info", "Testing DNS servers with domain: " .. domain)
    
    local results = {}
    
    for _, server in ipairs(servers) do
        log("info", "Testing DNS server: " .. server)
        
        local start_time = os.clock()
        local records, err = dns_query(domain, dns_record_types.A, server)
        local end_time = os.clock()
        
        local response_time = (end_time - start_time) * 1000 -- Convert to milliseconds
        
        if records and #records > 0 then
            log("info", "✅ " .. server .. " - Response time: " .. string.format("%.2f ms", response_time))
            table.insert(results, {
                server = server,
                status = "SUCCESS",
                response_time = response_time,
                records = #records
            })
        else
            log("warn", "❌ " .. server .. " - Failed: " .. (err or "unknown error"))
            table.insert(results, {
                server = server,
                status = "FAILED",
                error = err or "unknown error"
            })
        end
        
        sleep(1) -- Rate limiting
    end
    
    -- Sort by response time
    table.sort(results, function(a, b)
        if a.status == "SUCCESS" and b.status == "SUCCESS" then
            return a.response_time < b.response_time
        elseif a.status == "SUCCESS" then
            return true
        else
            return false
        end
    end)
    
    log("info", "=== DNS Server Performance ===")
    for i, result in ipairs(results) do
        if result.status == "SUCCESS" then
            log("info", string.format("%d. %s - %.2f ms (%d records)",
                i, result.server, result.response_time, result.records))
        else
            log("info", string.format("%d. %s - FAILED (%s)",
                i, result.server, result.error))
        end
    end
    
    return results
end

function reverse_dns_lookup(ip_address)
    log("info", "Performing reverse DNS lookup for: " .. ip_address)
    
    -- Convert IP to reverse DNS format (e.g., 8.8.8.8 -> 8.8.8.8.in-addr.arpa)
    local parts = {}
    for part in string.gmatch(ip_address, "[^%.]+") do
        table.insert(parts, 1, part) -- Insert at beginning to reverse
    end
    
    local reverse_domain = table.concat(parts, ".") .. ".in-addr.arpa"
    
    local records, err = dns_query(reverse_domain, dns_record_types.PTR)
    
    if records and #records > 0 then
        for _, record in ipairs(records) do
            log("info", ip_address .. " -> " .. record.data)
        end
        return records
    else
        log("warn", "Reverse DNS lookup failed: " .. (err or "no PTR record"))
        return nil
    end
end

function dns_zone_transfer_test(domain, name_servers)
    log("info", "Testing DNS zone transfer for: " .. domain)
    
    if not name_servers then
        -- First, get name servers for the domain
        local ns_records, err = dns_query(domain, dns_record_types.NS)
        if not ns_records then
            log("warn", "Could not get name servers for " .. domain)
            return
        end
        
        name_servers = {}
        for _, record in ipairs(ns_records) do
            table.insert(name_servers, record.data)
        end
    end
    
    for _, ns in ipairs(name_servers) do
        log("info", "Testing zone transfer from: " .. ns)
        
        -- Note: This is a simplified test
        -- Real zone transfer would use AXFR query type and TCP
        local conn, err = connect(ns, 53, "tcp")
        if conn then
            log("info", "Connected to " .. ns .. " - Zone transfer test would continue here")
            close(conn)
        else
            log("warn", "Could not connect to " .. ns .. ": " .. (err or "unknown error"))
        end
        
        sleep(1)
    end
end

-- Main execution
log("info", "GoCat DNS Resolver loaded!")

-- Example usage
local test_domain = "google.com"

log("info", "=== DNS Resolution Examples ===")

-- Basic A record lookup
resolve_domain(test_domain, {dns_record_types.A})

-- Multiple record types
resolve_domain(test_domain, {dns_record_types.A, dns_record_types.MX, dns_record_types.TXT})

-- DNS server performance test
dns_server_test(test_domain)

-- Reverse DNS lookup
reverse_dns_lookup("8.8.8.8")

log("info", "DNS resolver examples completed.")