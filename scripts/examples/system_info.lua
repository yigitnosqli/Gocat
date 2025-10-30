-- System Information Script for GoCat
-- Gathers and displays comprehensive system information

-- Get system information
function get_system_info()
    local info = {}
    
    -- Basic system info
    info.hostname = sys.hostname()
    info.platform = sys.platform()
    info.pid = sys.pid()
    info.pwd = sys.pwd()
    
    -- Environment variables
    info.env = {
        user = sys.env("USER"),
        home = sys.env("HOME"),
        path = sys.env("PATH"),
        shell = sys.env("SHELL"),
        term = sys.env("TERM"),
        lang = sys.env("LANG")
    }
    
    -- Current time
    info.timestamp = time.now()
    info.formatted_time = time.format(info.timestamp, "2006-01-02 15:04:05")
    
    return info
end

-- Display system information
function display_system_info(info)
    ui.cyan("╔══════════════════════════════════════════╗")
    ui.cyan("║         System Information               ║")
    ui.cyan("╚══════════════════════════════════════════╝")
    
    print("\n📊 Basic Information:")
    print(string.format("  Hostname: %s", info.hostname))
    print(string.format("  Platform: %s", info.platform))
    print(string.format("  Process ID: %d", info.pid))
    print(string.format("  Working Directory: %s", info.pwd))
    print(string.format("  Current Time: %s", info.formatted_time))
    
    print("\n🌍 Environment:")
    print(string.format("  User: %s", info.env.user or "N/A"))
    print(string.format("  Home: %s", info.env.home or "N/A"))
    print(string.format("  Shell: %s", info.env.shell or "N/A"))
    print(string.format("  Terminal: %s", info.env.term or "N/A"))
    print(string.format("  Language: %s", info.env.lang or "N/A"))
end

-- Directory scanner
function scan_directory(path)
    path = path or "."
    
    ui.info(string.format("\nScanning directory: %s", path))
    
    local files = sys.ls(path)
    if not files then
        ui.error("Failed to scan directory")
        return
    end
    
    local stats = {
        total = 0,
        files = 0,
        dirs = 0,
        total_size = 0
    }
    
    local file_list = {}
    local dir_list = {}
    
    for _, entry in ipairs(files) do
        stats.total = stats.total + 1
        
        if entry.isDir then
            stats.dirs = stats.dirs + 1
            table.insert(dir_list, entry.name)
        else
            stats.files = stats.files + 1
            stats.total_size = stats.total_size + (entry.size or 0)
            table.insert(file_list, {
                name = entry.name,
                size = entry.size or 0,
                mode = entry.mode or ""
            })
        end
    end
    
    -- Sort lists
    table.sort(dir_list)
    table.sort(file_list, function(a, b) return a.name < b.name end)
    
    -- Display results
    print(string.format("\n📁 Directory Statistics:"))
    print(string.format("  Total entries: %d", stats.total))
    print(string.format("  Directories: %d", stats.dirs))
    print(string.format("  Files: %d", stats.files))
    print(string.format("  Total size: %s", format_size(stats.total_size)))
    
    if #dir_list > 0 then
        ui.yellow("\n📂 Directories:")
        for i, dir in ipairs(dir_list) do
            if i <= 10 then  -- Show first 10
                print(string.format("  • %s/", dir))
            end
        end
        if #dir_list > 10 then
            print(string.format("  ... and %d more", #dir_list - 10))
        end
    end
    
    if #file_list > 0 then
        ui.green("\n📄 Files:")
        for i, file in ipairs(file_list) do
            if i <= 10 then  -- Show first 10
                print(string.format("  • %s (%s)", file.name, format_size(file.size)))
            end
        end
        if #file_list > 10 then
            print(string.format("  ... and %d more", #file_list - 10))
        end
    end
    
    return stats
end

-- Format file size
function format_size(bytes)
    if bytes < 1024 then
        return string.format("%d B", bytes)
    elseif bytes < 1024 * 1024 then
        return string.format("%.1f KB", bytes / 1024)
    elseif bytes < 1024 * 1024 * 1024 then
        return string.format("%.1f MB", bytes / (1024 * 1024))
    else
        return string.format("%.1f GB", bytes / (1024 * 1024 * 1024))
    end
end

-- Network interfaces information
function get_network_info()
    ui.cyan("\n🌐 Network Information:")
    
    -- Try to get network info via system commands
    if sys.platform() == "linux" or sys.platform() == "darwin" then
        -- Get hostname with domain
        local hostname_full = sys.exec("hostname", "-f")
        if hostname_full then
            print(string.format("  FQDN: %s", hostname_full:gsub("\n", "")))
        end
        
        -- Get IP addresses (simplified)
        print("  Local IPs:")
        print("    • 127.0.0.1 (localhost)")
        
        -- Try to connect to external service to get public IP
        local response = http.get("https://api.ipify.org?format=json")
        if response and response.status == 200 then
            local data = json.decode(response.body)
            if data and data.ip then
                print(string.format("  Public IP: %s", data.ip))
            end
        end
    end
end

-- File system usage
function check_disk_usage()
    ui.cyan("\n💾 Disk Usage:")
    
    local home = sys.env("HOME")
    if home then
        local home_stats = scan_directory(home)
        if home_stats then
            print(string.format("  Home directory: %d files, %d dirs", 
                home_stats.files, home_stats.dirs))
        end
    end
    
    -- Check current directory
    local current_stats = scan_directory(".")
    if current_stats then
        print(string.format("  Current directory: %d files, %d dirs", 
            current_stats.files, current_stats.dirs))
    end
end

-- Process information
function get_process_info()
    ui.cyan("\n⚙️ Process Information:")
    
    local pid = sys.pid()
    print(string.format("  Current PID: %d", pid))
    
    -- Get process start time
    local start_time = time.now()
    print(string.format("  Started: %s", time.format(start_time, "15:04:05")))
    
    -- Memory usage (if available)
    local mem_info = sys.env("MEMORY_LIMIT")
    if mem_info then
        print(string.format("  Memory Limit: %s", mem_info))
    end
end

-- Generate system report
function generate_report()
    local report = {}
    report.generated_at = os.date()
    report.system = get_system_info()
    
    -- Save report
    local filename = string.format("system_report_%s.json", os.date("%Y%m%d_%H%M%S"))
    local json_data = json.encode(report)
    
    file.write(filename, json.pretty(json_data))
    ui.success(string.format("\n📝 Report saved to: %s", filename))
    
    -- Also create a text version
    local text_filename = string.format("system_report_%s.txt", os.date("%Y%m%d_%H%M%S"))
    local text_content = "System Information Report\n"
    text_content = text_content .. string.format("Generated: %s\n", os.date())
    text_content = text_content .. string.format("Hostname: %s\n", report.system.hostname)
    text_content = text_content .. string.format("Platform: %s\n", report.system.platform)
    text_content = text_content .. string.format("Working Directory: %s\n", report.system.pwd)
    
    file.write(text_filename, text_content)
    ui.success(string.format("📝 Text report saved to: %s", text_filename))
end

-- Main function
function main()
    ui.cyan("╔══════════════════════════════════════════╗")
    ui.cyan("║      GoCat System Information v1.0      ║")
    ui.cyan("╚══════════════════════════════════════════╝")
    
    -- Gather and display system info
    local info = get_system_info()
    display_system_info(info)
    
    -- Network information
    get_network_info()
    
    -- Process information
    get_process_info()
    
    -- Disk usage
    check_disk_usage()
    
    -- Generate reports
    generate_report()
    
    ui.success("\n✅ System information gathering complete!")
end

-- Run if executed directly
if not ... then
    main()
end
