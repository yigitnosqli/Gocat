class Gocat < Formula
  desc "Modern netcat alternative written in Go with enhanced features"
  homepage "https://github.com/ibrahmsql/gocat"
  url "https://github.com/ibrahmsql/gocat/archive/refs/tags/v#{version}.tar.gz"
  sha256 "" # This will be updated when creating releases 
  license "MIT"
  head "https://github.com/ibrahmsql/gocat.git", branch: "main"

  depends_on "go" => :build

  def install
    # Build flags
    ldflags = %W[
      -s -w
      -X github.com/ibrahmsql/gocat/cmd.version=#{version}
      -X github.com/ibrahmsql/gocat/cmd.buildTime=#{Time.now.iso8601}
      -X github.com/ibrahmsql/gocat/cmd.gitCommit=homebrew-#{version}
      -X github.com/ibrahmsql/gocat/cmd.gitBranch=main
      -X github.com/ibrahmsql/gocat/cmd.builtBy=homebrew
    ]

    # Build the binary
    system "go", "build", *std_go_args(ldflags: ldflags)

    # Install man page
    (man1/"gocat.1").write <<~EOS
      .TH GOCAT 1 "#{Date.today.strftime("%B %Y")}" "GoCat #{version}" "User Commands"
      .SH NAME
      gocat \- modern netcat alternative written in Go
      .SH SYNOPSIS
      .B gocat
      [\fIOPTIONS\fR] \fICOMMAND\fR [\fIARGS\fR]
      .SH DESCRIPTION
      GoCat is a modern, feature-rich alternative to netcat written in Go. It provides enhanced functionality for network communication, debugging, and penetration testing.
      .SH OPTIONS
      .TP
      \fB\-h, \-\-help\fR
      Show help message
      .TP
      \fB\-v, \-\-version\fR
      Show version information
      .SH COMMANDS
      .TP
      \fBlisten\fR \fIPORT\fR
      Listen on specified port
      .TP
      \fBconnect\fR \fIHOST\fR \fIPORT\fR
      Connect to specified host and port
      .SH EXAMPLES
      .TP
      Listen on port 8080:
      .B gocat listen 8080
      .TP
      Connect to example.com on port 80:
      .B gocat connect example.com 80
      .SH AUTHOR
      Written by Ibrahim SQL.
      .SH REPORTING BUGS
      Report bugs to: https://github.com/ibrahmsql/gocat/issues
      .SH COPYRIGHT
      Copyright © 2025 ibrahimsql. License MIT.
    EOS

    # Install bash completion
    (bash_completion/"gocat").write <<~EOS
      _gocat() {
          local cur prev opts
          COMPREPLY=()
          cur="${COMP_WORDS[COMP_CWORD]}"
          prev="${COMP_WORDS[COMP_CWORD-1]}"
          opts="--help --version --debug --timeout --buffer-size listen connect"

          case "${prev}" in
              listen)
                  # Port completion for listen command
                  COMPREPLY=( $(compgen -W "8080 9000 3000 8000" -- ${cur}) )
                  return 0
                  ;;
              connect)
                  # Hostname completion for connect command
                  COMPREPLY=( $(compgen -W "localhost 127.0.0.1 example.com" -- ${cur}) )
                  return 0
                  ;;
              --timeout|--buffer-size)
                  # Numeric values
                  COMPREPLY=( $(compgen -W "30 60 120 1024 2048 4096" -- ${cur}) )
                  return 0
                  ;;
          esac

          COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
          return 0
      }
      complete -F _gocat gocat
    EOS

    # Install zsh completion
    (zsh_completion/"_gocat").write <<~EOS
      #compdef gocat

      _gocat() {
          local context state line
          typeset -A opt_args

          _arguments -C \
              '(--help -h)'{--help,-h}'[Show help message]' \
              '(--version -v)'{--version,-v}'[Show version information]' \
              '--debug[Enable debug mode]' \
              '--timeout[Set connection timeout]:timeout:(30 60 120)' \
              '--buffer-size[Set buffer size]:size:(1024 2048 4096)' \
              '1: :_gocat_commands' \
              '*:: :->args'

          case $state in
              args)
                  case $words[1] in
                      listen)
                          _arguments \
                              '1:port:(8080 9000 3000 8000)'
                          ;;
                      connect)
                          _arguments \
                              '1:host:(localhost 127.0.0.1 example.com)' \
                              '2:port:(80 443 22 21)'
                          ;;
                  esac
                  ;;
          esac
      }

      _gocat_commands() {
          local commands
          commands=(
              'listen:Listen on specified port'
              'connect:Connect to specified host and port'
          )
          _describe 'commands' commands
      }

      _gocat "$@"
    EOS
  end

  test do
    # Test version output
    assert_match version.to_s, shell_output("#{bin}/gocat --version")
    
    # Test help output
    assert_match "Usage:", shell_output("#{bin}/gocat --help")
    
    # Test that binary exists and is executable
    assert_predicate bin/"gocat", :exist?
    assert_predicate bin/"gocat", :executable?
  end
end