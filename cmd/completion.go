package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fatih/color"
	"github.com/ibrahmsql/gocat/internal/logger"
	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for various shells.

The completion script can be sourced to enable tab completion for gocat commands.`,
	Example: `  # Bash completion
  gocat completion bash > ~/.gocat-completion.bash
  echo "source ~/.gocat-completion.bash" >> ~/.bashrc

  # Zsh completion
  gocat completion zsh > ~/.gocat-completion.zsh
  echo "source ~/.gocat-completion.zsh" >> ~/.zshrc

  # Fish completion
  gocat completion fish > ~/.config/fish/completions/gocat.fish

  # PowerShell completion
  gocat completion powershell > gocat.ps1`,
}

var completionInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Auto-install shell completions",
	Long: `Automatically detect your shell and install completions.

This command will:
1. Detect your current shell
2. Generate appropriate completion script
3. Install it in the correct location
4. Update your shell configuration`,
	Run: autoInstallCompletion,
}

var completionBashCmd = &cobra.Command{
	Use:   "bash",
	Short: "Generate bash completion script",
	Run: func(cmd *cobra.Command, args []string) {
		rootCmd.GenBashCompletion(os.Stdout)
	},
}

var completionZshCmd = &cobra.Command{
	Use:   "zsh", 
	Short: "Generate zsh completion script",
	Run: func(cmd *cobra.Command, args []string) {
		rootCmd.GenZshCompletion(os.Stdout)
	},
}

var completionFishCmd = &cobra.Command{
	Use:   "fish",
	Short: "Generate fish completion script",
	Run: func(cmd *cobra.Command, args []string) {
		rootCmd.GenFishCompletion(os.Stdout, true)
	},
}

var completionPowerShellCmd = &cobra.Command{
	Use:   "powershell",
	Short: "Generate PowerShell completion script",
	Run: func(cmd *cobra.Command, args []string) {
		rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
	
	completionCmd.AddCommand(completionInstallCmd)
	completionCmd.AddCommand(completionBashCmd)
	completionCmd.AddCommand(completionZshCmd)
	completionCmd.AddCommand(completionFishCmd)
	completionCmd.AddCommand(completionPowerShellCmd)
}

func autoInstallCompletion(cmd *cobra.Command, args []string) {
	// Detect shell
	shell := detectShell()
	if shell == "" {
		logger.Error("Could not detect shell. Please specify manually.")
		return
	}

	logger.Info("Detected shell: %s", shell)

	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logger.Error("Failed to get home directory: %v", err)
		return
	}

	var completionFile string
	var configFile string
	var sourceCmd string

	switch shell {
	case "bash":
		completionFile = filepath.Join(homeDir, ".gocat-completion.bash")
		configFile = filepath.Join(homeDir, ".bashrc")
		sourceCmd = fmt.Sprintf("source %s", completionFile)
		
		// Generate bash completion
		file, err := os.Create(completionFile)
		if err != nil {
			logger.Error("Failed to create completion file: %v", err)
			return
		}
		defer file.Close()
		
		if err := rootCmd.GenBashCompletion(file); err != nil {
			logger.Error("Failed to generate bash completion: %v", err)
			return
		}

	case "zsh":
		// For zsh, we need to add to fpath
		completionDir := filepath.Join(homeDir, ".zsh", "completions")
		os.MkdirAll(completionDir, 0755)
		
		completionFile = filepath.Join(completionDir, "_gocat")
		configFile = filepath.Join(homeDir, ".zshrc")
		
		// Add fpath and compinit if not present
		sourceCmd = fmt.Sprintf(`
# GoCat completion
fpath=(%s $fpath)
autoload -Uz compinit && compinit`, completionDir)
		
		// Generate zsh completion
		file, err := os.Create(completionFile)
		if err != nil {
			logger.Error("Failed to create completion file: %v", err)
			return
		}
		defer file.Close()
		
		if err := rootCmd.GenZshCompletion(file); err != nil {
			logger.Error("Failed to generate zsh completion: %v", err)
			return
		}

	case "fish":
		completionDir := filepath.Join(homeDir, ".config", "fish", "completions")
		os.MkdirAll(completionDir, 0755)
		
		completionFile = filepath.Join(completionDir, "gocat.fish")
		
		// Generate fish completion
		file, err := os.Create(completionFile)
		if err != nil {
			logger.Error("Failed to create completion file: %v", err)
			return
		}
		defer file.Close()
		
		if err := rootCmd.GenFishCompletion(file, true); err != nil {
			logger.Error("Failed to generate fish completion: %v", err)
			return
		}
		
		color.Green("✅ Fish completion installed to %s", completionFile)
		color.Yellow("Fish will automatically load completions from this directory.")
		return

	case "powershell":
		completionFile = filepath.Join(homeDir, "Documents", "WindowsPowerShell", "gocat.ps1")
		configFile = filepath.Join(homeDir, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1")
		sourceCmd = fmt.Sprintf(". %s", completionFile)
		
		// Generate PowerShell completion
		file, err := os.Create(completionFile)
		if err != nil {
			logger.Error("Failed to create completion file: %v", err)
			return
		}
		defer file.Close()
		
		if err := rootCmd.GenPowerShellCompletionWithDesc(file); err != nil {
			logger.Error("Failed to generate PowerShell completion: %v", err)
			return
		}

	default:
		logger.Error("Unsupported shell: %s", shell)
		return
	}

	// Add source command to shell config if needed (except fish)
	if configFile != "" {
		if err := addToShellConfig(configFile, sourceCmd); err != nil {
			logger.Error("Failed to update shell config: %v", err)
			color.Yellow("Please add the following line to your %s manually:", configFile)
			fmt.Println(sourceCmd)
		} else {
			color.Green("✅ Completion installed successfully!")
			color.Yellow("Please restart your shell or run:")
			fmt.Printf("   source %s\n", configFile)
		}
	}
}

func detectShell() string {
	// First try SHELL environment variable
	if shellEnv := os.Getenv("SHELL"); shellEnv != "" {
		shellName := filepath.Base(shellEnv)
		if isValidShell(shellName) {
			return shellName
		}
	}

	// On Windows, check for PowerShell
	if runtime.GOOS == "windows" {
		if _, err := exec.LookPath("powershell.exe"); err == nil {
			return "powershell"
		}
	}

	// Try to detect from parent process
	ppid := os.Getppid()
	if ppid > 0 {
		// Try ps command
		out, err := exec.Command("ps", "-p", fmt.Sprintf("%d", ppid), "-o", "comm=").Output()
		if err == nil {
			shellName := strings.TrimSpace(string(out))
			shellName = filepath.Base(shellName)
			if isValidShell(shellName) {
				return shellName
			}
		}
	}

	// Try common shells
	shells := []string{"bash", "zsh", "fish", "sh"}
	for _, shell := range shells {
		if _, err := exec.LookPath(shell); err == nil {
			return shell
		}
	}

	return ""
}

func isValidShell(name string) bool {
	validShells := []string{"bash", "zsh", "fish", "sh", "powershell", "pwsh"}
	for _, valid := range validShells {
		if name == valid {
			return true
		}
	}
	return false
}

func addToShellConfig(configFile, sourceCmd string) error {
	// Read existing config
	content, err := os.ReadFile(configFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Check if already added
	if strings.Contains(string(content), "GoCat completion") {
		logger.Info("Completion already configured in %s", configFile)
		return nil
	}

	// Append to config
	file, err := os.OpenFile(configFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Add newline if file doesn't end with one
	if len(content) > 0 && content[len(content)-1] != '\n' {
		file.WriteString("\n")
	}

	// Write completion source
	file.WriteString("\n" + sourceCmd + "\n")

	return nil
}
