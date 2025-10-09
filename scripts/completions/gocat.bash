#!/usr/bin/env bash
# Bash completion for gocat

_gocat_completions() {
    local cur prev opts commands
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    
    # Main commands
    commands="connect listen scan broker chat tunnel dns-tunnel multi-listen proxy convert transfer script version help"
    
    # Global options
    opts="--ipv4 --ipv6 --udp --sctp --listen --keep-open --verbose --debug --quiet --ssl --ssl-cert --ssl-key --ssl-verify --wait --proxy --rate-limit --allow --deny --config --theme --json --no-color --help --version"
    
    # If we're completing the first argument (command)
    if [ $COMP_CWORD -eq 1 ]; then
        COMPREPLY=( $(compgen -W "${commands}" -- ${cur}) )
        return 0
    fi
    
    # Get the command
    local command="${COMP_WORDS[1]}"
    
    # Command-specific completions
    case "${command}" in
        connect)
            case "${prev}" in
                --shell|-s)
                    COMPREPLY=( $(compgen -c -- ${cur}) )
                    return 0
                    ;;
                --timeout|-t|--wait|-w)
                    COMPREPLY=( $(compgen -W "5s 10s 30s 1m 5m" -- ${cur}) )
                    return 0
                    ;;
                --retry|-r)
                    COMPREPLY=( $(compgen -W "1 3 5 10" -- ${cur}) )
                    return 0
                    ;;
                *)
                    COMPREPLY=( $(compgen -W "--shell --timeout --retry --keep-alive --proxy --ssl ${opts}" -- ${cur}) )
                    return 0
                    ;;
            esac
            ;;
        listen)
            case "${prev}" in
                --exec|-e)
                    COMPREPLY=( $(compgen -c -- ${cur}) )
                    return 0
                    ;;
                --bind|-b)
                    COMPREPLY=( $(compgen -W "0.0.0.0 127.0.0.1 ::" -- ${cur}) )
                    return 0
                    ;;
                --max-conn|-m)
                    COMPREPLY=( $(compgen -W "10 50 100 500 1000" -- ${cur}) )
                    return 0
                    ;;
                *)
                    COMPREPLY=( $(compgen -W "--exec --interactive --bind --max-conn --keep-alive --ssl --ssl-cert --ssl-key ${opts}" -- ${cur}) )
                    return 0
                    ;;
            esac
            ;;
        scan)
            case "${prev}" in
                --timeout|-t)
                    COMPREPLY=( $(compgen -W "1s 3s 5s 10s" -- ${cur}) )
                    return 0
                    ;;
                --concurrent|-c)
                    COMPREPLY=( $(compgen -W "10 50 100 500 1000" -- ${cur}) )
                    return 0
                    ;;
                --output|-o)
                    COMPREPLY=( $(compgen -W "text json xml" -- ${cur}) )
                    return 0
                    ;;
                *)
                    COMPREPLY=( $(compgen -W "--timeout --concurrent --output --udp --tcp ${opts}" -- ${cur}) )
                    return 0
                    ;;
            esac
            ;;
        proxy)
            case "${prev}" in
                --listen)
                    COMPREPLY=( $(compgen -W ":8080 :8000 :3000 0.0.0.0:8080" -- ${cur}) )
                    return 0
                    ;;
                --lb-algorithm)
                    COMPREPLY=( $(compgen -W "round-robin least-connections random ip-hash" -- ${cur}) )
                    return 0
                    ;;
                *)
                    COMPREPLY=( $(compgen -W "--listen --target --backends --lb-algorithm --health-check --ssl --ssl-cert --ssl-key ${opts}" -- ${cur}) )
                    return 0
                    ;;
            esac
            ;;
        tunnel)
            case "${prev}" in
                --key)
                    COMPREPLY=( $(compgen -f -- ${cur}) )
                    return 0
                    ;;
                *)
                    COMPREPLY=( $(compgen -W "--ssh --local --remote --dynamic --reverse --key ${opts}" -- ${cur}) )
                    return 0
                    ;;
            esac
            ;;
        convert)
            case "${prev}" in
                --from)
                    COMPREPLY=( $(compgen -W "tcp: udp: http: ws:" -- ${cur}) )
                    return 0
                    ;;
                --to)
                    COMPREPLY=( $(compgen -W "tcp: udp: http: ws:" -- ${cur}) )
                    return 0
                    ;;
                *)
                    COMPREPLY=( $(compgen -W "--from --to ${opts}" -- ${cur}) )
                    return 0
                    ;;
            esac
            ;;
        script)
            case "${prev}" in
                script)
                    COMPREPLY=( $(compgen -f -X '!*.lua' -- ${cur}) )
                    return 0
                    ;;
                *)
                    COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
                    return 0
                    ;;
            esac
            ;;
        *)
            COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
            return 0
            ;;
    esac
}

complete -F _gocat_completions gocat
