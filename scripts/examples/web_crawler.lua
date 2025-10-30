-- Web Crawler Script for GoCat
-- Crawls websites and extracts information

local visited = {}
local queue = {}
local max_depth = 3
local max_pages = 100
local pages_crawled = 0

-- Extract links from HTML
function extract_links(html, base_url)
    local links = {}
    
    -- Extract href attributes
    for link in string.gmatch(html, 'href="([^"]+)"') do
        if not string.match(link, "^#") then  -- Skip anchors
            if string.match(link, "^http") then
                table.insert(links, link)
            elseif string.match(link, "^/") then
                table.insert(links, base_url .. link)
            end
        end
    end
    
    -- Extract src attributes (images, scripts)
    for link in string.gmatch(html, 'src="([^"]+)"') do
        if string.match(link, "^http") then
            table.insert(links, link)
        elseif string.match(link, "^/") then
            table.insert(links, base_url .. link)
        end
    end
    
    return links
end

-- Extract metadata from HTML
function extract_metadata(html)
    local metadata = {}
    
    -- Extract title
    local title = string.match(html, "<title>(.-)</title>")
    if title then
        metadata.title = title
    end
    
    -- Extract meta description
    local description = string.match(html, '<meta name="description" content="([^"]+)"')
    if description then
        metadata.description = description
    end
    
    -- Extract meta keywords
    local keywords = string.match(html, '<meta name="keywords" content="([^"]+)"')
    if keywords then
        metadata.keywords = keywords
    end
    
    return metadata
end

-- Crawl a single URL
function crawl_url(url, depth)
    if visited[url] or pages_crawled >= max_pages then
        return
    end
    
    visited[url] = true
    pages_crawled = pages_crawled + 1
    
    ui.info(string.format("Crawling [%d/%d]: %s (depth: %d)", 
        pages_crawled, max_pages, url, depth))
    
    -- Fetch the page
    local response = http.get(url)
    if not response or response.status ~= 200 then
        ui.warn(string.format("Failed to fetch: %s", url))
        return
    end
    
    local html = response.body
    
    -- Extract metadata
    local metadata = extract_metadata(html)
    if metadata.title then
        ui.green(string.format("  Title: %s", metadata.title))
    end
    if metadata.description then
        ui.cyan(string.format("  Description: %s", metadata.description:sub(1, 100)))
    end
    
    -- Extract and process links
    if depth < max_depth then
        local base_url = string.match(url, "^(https?://[^/]+)")
        local links = extract_links(html, base_url)
        
        ui.info(string.format("  Found %d links", #links))
        
        for _, link in ipairs(links) do
            if not visited[link] then
                table.insert(queue, {url = link, depth = depth + 1})
            end
        end
    end
    
    -- Small delay to be polite
    time.sleep(0.5)
end

-- Process the crawl queue
function process_queue()
    while #queue > 0 and pages_crawled < max_pages do
        local item = table.remove(queue, 1)
        crawl_url(item.url, item.depth)
    end
end

-- Analyze crawl results
function analyze_results()
    ui.cyan("\n╔══════════════════════════════════════════╗")
    ui.cyan("║           Crawl Statistics              ║")
    ui.cyan("╚══════════════════════════════════════════╝")
    
    print(string.format("\nPages crawled: %d", pages_crawled))
    print(string.format("Unique URLs visited: %d", table_length(visited)))
    print(string.format("URLs in queue: %d", #queue))
    
    -- Save results
    local results = {
        timestamp = os.date(),
        pages_crawled = pages_crawled,
        urls = {}
    }
    
    for url, _ in pairs(visited) do
        table.insert(results.urls, url)
    end
    
    local json_data = json.encode(results)
    local filename = string.format("crawl_%s.json", os.date("%Y%m%d_%H%M%S"))
    file.write(filename, json_data)
    
    ui.success(string.format("\nResults saved to %s", filename))
end

-- Helper function to get table length
function table_length(t)
    local count = 0
    for _ in pairs(t) do
        count = count + 1
    end
    return count
end

-- Main crawler function
function crawl(start_url, options)
    options = options or {}
    max_depth = options.max_depth or max_depth
    max_pages = options.max_pages or max_pages
    
    ui.cyan("╔══════════════════════════════════════════╗")
    ui.cyan("║          GoCat Web Crawler v1.0         ║")
    ui.cyan("╚══════════════════════════════════════════╝")
    print()
    
    ui.info(string.format("Starting crawl from: %s", start_url))
    ui.info(string.format("Max depth: %d, Max pages: %d", max_depth, max_pages))
    print()
    
    -- Initialize queue with start URL
    table.insert(queue, {url = start_url, depth = 0})
    
    -- Start crawling
    local start_time = time.now()
    process_queue()
    local elapsed = time.since(start_time)
    
    -- Show results
    analyze_results()
    ui.success(string.format("\nCrawl completed in %.2f seconds", elapsed))
end

-- Main function
function main(args)
    local url = args and args[1] or "https://example.com"
    
    local options = {
        max_depth = 3,
        max_pages = 50
    }
    
    crawl(url, options)
end

-- Run if executed directly
if not ... then
    main()
end
