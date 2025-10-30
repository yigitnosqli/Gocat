-- File Manager Script for GoCat
-- Advanced file operations and management

-- File operations menu
function file_menu()
    ui.cyan("╔══════════════════════════════════════════╗")
    ui.cyan("║        GoCat File Manager v1.0          ║")
    ui.cyan("╚══════════════════════════════════════════╝")
    
    print("\nFile Operations:")
    print("1. List files")
    print("2. Read file")
    print("3. Write file")
    print("4. Copy file")
    print("5. Move file")
    print("6. Delete file")
    print("7. Search files")
    print("8. Backup directory")
end

-- List files with details
function list_files(path)
    path = path or "."
    
    ui.info(string.format("\n📁 Listing: %s", path))
    
    local files = sys.ls(path)
    if not files then
        ui.error("Failed to list directory")
        return
    end
    
    -- Separate files and directories
    local dirs = {}
    local regular_files = {}
    
    for _, entry in ipairs(files) do
        if entry.isDir then
            table.insert(dirs, entry)
        else
            table.insert(regular_files, entry)
        end
    end
    
    -- Sort
    table.sort(dirs, function(a, b) return a.name < b.name end)
    table.sort(regular_files, function(a, b) return a.name < b.name end)
    
    -- Display directories first
    if #dirs > 0 then
        ui.yellow("\nDirectories:")
        for _, dir in ipairs(dirs) do
            print(string.format("  📂 %s/", dir.name))
        end
    end
    
    -- Display files
    if #regular_files > 0 then
        ui.green("\nFiles:")
        for _, f in ipairs(regular_files) do
            local icon = get_file_icon(f.name)
            local size_str = format_size(f.size or 0)
            print(string.format("  %s %s (%s)", icon, f.name, size_str))
        end
    end
    
    print(string.format("\nTotal: %d items (%d dirs, %d files)", 
        #dirs + #regular_files, #dirs, #regular_files))
end

-- Get file icon based on extension
function get_file_icon(filename)
    local ext = string.match(filename, "%.([^.]+)$")
    if not ext then return "📄" end
    
    ext = string.lower(ext)
    
    local icons = {
        -- Code files
        lua = "🌙",
        go = "🐹",
        py = "🐍",
        js = "📜",
        html = "🌐",
        css = "🎨",
        json = "📊",
        xml = "📋",
        
        -- Documents
        txt = "📝",
        md = "📖",
        pdf = "📕",
        doc = "📘",
        
        -- Archives
        zip = "📦",
        tar = "📦",
        gz = "📦",
        
        -- Images
        png = "🖼️",
        jpg = "🖼️",
        jpeg = "🖼️",
        gif = "🎞️",
        
        -- Config
        conf = "⚙️",
        cfg = "⚙️",
        ini = "⚙️",
        yaml = "⚙️",
        yml = "⚙️",
        
        -- Scripts
        sh = "🔧",
        bash = "🔧",
        
        -- Data
        sql = "🗄️",
        db = "🗄️",
        csv = "📊"
    }
    
    return icons[ext] or "📄"
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

-- Read and display file content
function read_file(filepath)
    ui.info(string.format("\n📖 Reading: %s", filepath))
    
    local content = file.read(filepath)
    if not content then
        ui.error("Failed to read file")
        return
    end
    
    local stats = file.stat(filepath)
    if stats then
        print(string.format("Size: %s", format_size(stats.size)))
        print(string.format("Modified: %s", time.format(stats.modTime, "2006-01-02 15:04:05")))
    end
    
    print("\n--- Content ---")
    
    -- Limit display for large files
    if string.len(content) > 5000 then
        print(string.sub(content, 1, 5000))
        ui.warn(string.format("\n... (truncated, showing first 5000 of %d bytes)", string.len(content)))
    else
        print(content)
    end
    
    print("\n--- End of file ---")
end

-- Write content to file
function write_file(filepath, content)
    ui.info(string.format("\n✍️ Writing to: %s", filepath))
    
    local success = file.write(filepath, content)
    if success then
        ui.success(string.format("Successfully wrote %d bytes", string.len(content)))
        
        -- Verify
        local verify = file.read(filepath)
        if verify == content then
            ui.success("✅ Write verified")
        else
            ui.warn("⚠️ Verification failed")
        end
    else
        ui.error("Failed to write file")
    end
    
    return success
end

-- Copy file with progress
function copy_file(src, dst)
    ui.info(string.format("\n📋 Copying: %s -> %s", src, dst))
    
    -- Check source exists
    if not file.exists(src) then
        ui.error("Source file does not exist")
        return false
    end
    
    -- Check destination
    if file.exists(dst) then
        ui.warn("Destination already exists, will overwrite")
    end
    
    local success = file.copy(src, dst)
    if success then
        ui.success("✅ File copied successfully")
        
        -- Verify sizes match
        local src_stat = file.stat(src)
        local dst_stat = file.stat(dst)
        
        if src_stat and dst_stat and src_stat.size == dst_stat.size then
            ui.success(string.format("✅ Verified: %s copied", format_size(src_stat.size)))
        end
    else
        ui.error("Failed to copy file")
    end
    
    return success
end

-- Search for files
function search_files(pattern, path)
    path = path or "."
    
    ui.info(string.format("\n🔍 Searching for '%s' in %s", pattern, path))
    
    local matches = {}
    local function search_recursive(dir)
        local files = sys.ls(dir)
        if not files then return end
        
        for _, entry in ipairs(files) do
            local fullpath = file.join(dir, entry.name)
            
            -- Check if name matches pattern
            if string.find(entry.name, pattern) then
                table.insert(matches, {
                    path = fullpath,
                    name = entry.name,
                    isDir = entry.isDir,
                    size = entry.size
                })
            end
            
            -- Recurse into directories
            if entry.isDir and entry.name ~= "." and entry.name ~= ".." then
                search_recursive(fullpath)
            end
        end
    end
    
    search_recursive(path)
    
    -- Display results
    if #matches > 0 then
        ui.success(string.format("\nFound %d matches:", #matches))
        for _, match in ipairs(matches) do
            local icon = match.isDir and "📂" or get_file_icon(match.name)
            local size_str = match.isDir and "" or string.format(" (%s)", format_size(match.size or 0))
            print(string.format("  %s %s%s", icon, match.path, size_str))
        end
    else
        ui.warn("No matches found")
    end
    
    return matches
end

-- Backup directory
function backup_directory(src_dir, backup_name)
    src_dir = src_dir or "."
    backup_name = backup_name or string.format("backup_%s.tar", os.date("%Y%m%d_%H%M%S"))
    
    ui.info(string.format("\n💾 Creating backup of %s", src_dir))
    
    -- Get list of files
    local files = sys.ls(src_dir)
    if not files then
        ui.error("Failed to read directory")
        return false
    end
    
    local file_count = 0
    local total_size = 0
    
    -- Count files and calculate size
    for _, entry in ipairs(files) do
        if not entry.isDir then
            file_count = file_count + 1
            total_size = total_size + (entry.size or 0)
        end
    end
    
    ui.info(string.format("Backing up %d files (%s)", file_count, format_size(total_size)))
    
    -- Create backup (simplified - just copy files to backup directory)
    local backup_dir = string.format("backup_%s", os.date("%Y%m%d_%H%M%S"))
    sys.mkdir(backup_dir)
    
    local backed_up = 0
    for _, entry in ipairs(files) do
        if not entry.isDir then
            local src = file.join(src_dir, entry.name)
            local dst = file.join(backup_dir, entry.name)
            
            if file.copy(src, dst) then
                backed_up = backed_up + 1
                ui.progress(backed_up, file_count, string.format("Backing up %s", entry.name))
            end
        end
    end
    
    ui.success(string.format("\n✅ Backup complete: %d/%d files backed up to %s", 
        backed_up, file_count, backup_dir))
    
    return true
end

-- Interactive file manager
function interactive_mode()
    file_menu()
    
    -- Demo operations
    ui.info("\n🎯 Running file management demonstrations...")
    
    -- List current directory
    ui.yellow("\n=== Directory Listing ===")
    list_files(".")
    
    -- Create test file
    ui.yellow("\n=== File Operations ===")
    local test_file = "test_gocat.txt"
    local test_content = "Hello from GoCat File Manager!\nTime: " .. os.date()
    write_file(test_file, test_content)
    
    -- Read it back
    read_file(test_file)
    
    -- Copy file
    local copy_name = "test_gocat_copy.txt"
    copy_file(test_file, copy_name)
    
    -- Search for files
    ui.yellow("\n=== File Search ===")
    search_files("%.lua$", ".")
    
    -- Clean up
    ui.info("\n🧹 Cleaning up test files...")
    file.delete(test_file)
    file.delete(copy_name)
    ui.success("✅ Cleanup complete")
end

-- Main function
function main(args)
    if args and args[1] then
        local cmd = args[1]
        
        if cmd == "list" then
            list_files(args[2] or ".")
        elseif cmd == "read" then
            if args[2] then
                read_file(args[2])
            else
                ui.error("Please specify a file to read")
            end
        elseif cmd == "search" then
            if args[2] then
                search_files(args[2], args[3] or ".")
            else
                ui.error("Please specify a search pattern")
            end
        elseif cmd == "backup" then
            backup_directory(args[2] or ".", args[3])
        else
            ui.error(string.format("Unknown command: %s", cmd))
        end
    else
        -- Interactive mode
        interactive_mode()
    end
end

-- Run if executed directly
if not ... then
    main()
end
