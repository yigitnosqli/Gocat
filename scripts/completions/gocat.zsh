#compdef gocat

_gocat() {
    local -a commands
    commands=(
        'connect:Connect to a remote host'
        'listen:Listen for incoming connections'
        'scan:Scan ports on target host'
        'broker:Start connection broker'
        'chat:Start chat server'
        'tunnel:Create SSH tunnels'
        'dns-tunnel:Create DNS tunnels'
        'multi-listen:Listen on multiple ports'
        'proxy:Start HTTP reverse proxy'
        'convert:Convert between protocols'
        'transfer:Transfer files'
        'script:Execute Lua script'
        'version:Show version information'
        'help:Show help information'
    )

    local -a global_opts
    global_opts=(
        '(-4 --ipv4)'{-4,--ipv4}'[Use IPv4 only]'
        '(-6 --ipv6)'{-6,--ipv6}'[Use IPv6 only]'
        '(-u --udp)'{-u,--udp}'[Use UDP instead of TCP]'
        '--sctp[Use SCTP protocol]'
        '(-l --listen)'{-l,--listen}'[Listen mode]'
        '(-k --keep-open)'{-k,--keep-open}'[Accept multiple connections]'
        '(-v --verbose)'{-v,--verbose}'[Verbose output]'
        '--debug[Debug mode]'
        '(-q --quiet)'{-q,--quiet}'[Quiet mode]'
        '--ssl[Use SSL/TLS]'
        '--ssl-cert[SSL certificate file]:file:_files'
        '--ssl-key[SSL private key file]:file:_files'
        '--ssl-verify[Verify SSL certificates]'
        '(-w --wait)'{-w,--wait}'[Connection timeout]:duration:'
        '--proxy[Use proxy]:url:'
        '--rate-limit[Rate limit]:rate:'
        '--allow[Allow IP addresses]:ip:'
        '--deny[Deny IP addresses]:ip:'
        '--config[Configuration file]:file:_files'
        '--theme[Color theme file]:file:_files'
        '--json[JSON output]'
        '--no-color[Disable colors]'
        '(-h --help)'{-h,--help}'[Show help]'
        '--version[Show version]'
    )

    _arguments -C \
        "1: :->command" \
        "*::arg:->args" \
        $global_opts

    case $state in
        command)
            _describe 'command' commands
            ;;
        args)
            case $words[1] in
                connect)
                    _arguments \
                        '(-s --shell)'{-s,--shell}'[Shell to use]:shell:_command_names' \
                        '(-t --timeout)'{-t,--timeout}'[Connection timeout]:duration:' \
                        '(-r --retry)'{-r,--retry}'[Retry attempts]:count:' \
                        '--keep-alive[Enable keep-alive]' \
                        '1:host:_hosts' \
                        '2:port:' \
                        $global_opts
                    ;;
                listen)
                    _arguments \
                        '(-e --exec)'{-e,--exec}'[Execute command]:command:_command_names' \
                        '(-i --interactive)'{-i,--interactive}'[Interactive mode]' \
                        '(-b --bind)'{-b,--bind}'[Bind address]:address:' \
                        '(-m --max-conn)'{-m,--max-conn}'[Max connections]:count:' \
                        '1:port:' \
                        $global_opts
                    ;;
                scan)
                    _arguments \
                        '(-t --timeout)'{-t,--timeout}'[Scan timeout]:duration:' \
                        '(-c --concurrent)'{-c,--concurrent}'[Concurrent scans]:count:' \
                        '(-o --output)'{-o,--output}'[Output format]:format:(text json xml)' \
                        '1:host:_hosts' \
                        '2:ports:' \
                        $global_opts
                    ;;
                proxy)
                    _arguments \
                        '--listen[Listen address]:address:' \
                        '--target[Target URL]:url:' \
                        '--backends[Backend URLs]:urls:' \
                        '--lb-algorithm[Load balancing algorithm]:algorithm:(round-robin least-connections random ip-hash)' \
                        '--health-check[Health check path]:path:' \
                        $global_opts
                    ;;
                tunnel)
                    _arguments \
                        '--ssh[SSH server]:server:' \
                        '--local[Local port]:port:' \
                        '--remote[Remote address]:address:' \
                        '--dynamic[Dynamic SOCKS port]:port:' \
                        '--reverse[Reverse forwarding]' \
                        '--key[SSH key file]:file:_files' \
                        $global_opts
                    ;;
                convert)
                    _arguments \
                        '--from[Source protocol]:protocol:' \
                        '--to[Target protocol]:protocol:' \
                        $global_opts
                    ;;
                script)
                    _arguments \
                        '1:script:_files -g "*.lua"' \
                        $global_opts
                    ;;
                *)
                    _arguments $global_opts
                    ;;
            esac
            ;;
    esac
}

_gocat "$@"
