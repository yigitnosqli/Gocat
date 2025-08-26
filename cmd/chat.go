package cmd

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/ibrahmsql/gocat/internal/logger"
	"github.com/spf13/cobra"
)

var (
	chatPort     string
	chatClients  map[string]*ChatClient
	chatMutex    sync.RWMutex
	chatMaxConns int
	chatRoomName string
)

type ChatClient struct {
	Conn     net.Conn
	Nickname string
	JoinTime time.Time
}

var chatCmd = &cobra.Command{
	Use:   "chat [port]",
	Short: "Start a chat server mode",
	Long: `Start GoCat in chat server mode. This creates a simple chat room where
multiple clients can connect and exchange messages with nicknames.`,
	Args: cobra.ExactArgs(1),
	Run:  runChat,
}

func init() {
	rootCmd.AddCommand(chatCmd)
	chatCmd.Flags().IntVarP(&chatMaxConns, "max-conns", "m", 20, "Maximum number of concurrent chat connections")
	chatCmd.Flags().StringVarP(&chatRoomName, "room", "r", "GoCat-Room", "Chat room name")
	chatClients = make(map[string]*ChatClient)
}

func runChat(cmd *cobra.Command, args []string) {
	chatPort = args[0]

	// Override with global flags if set
	if globalMaxConns, _ := cmd.Root().PersistentFlags().GetInt("max-conns"); globalMaxConns > 0 {
		chatMaxConns = globalMaxConns
	}

	logger.Info("Starting chat server '%s' on port %s (max connections: %d)", chatRoomName, chatPort, chatMaxConns)

	if err := startChatServer(chatPort); err != nil {
		logger.Fatal("Chat server error: %v", err)
	}
}

func startChatServer(port string) error {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("failed to start chat server: %w", err)
	}
	defer listener.Close()

	logger.Info("Chat server '%s' listening on :%s", chatRoomName, port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Error("Failed to accept chat connection: %v", err)
			continue
		}

		chatMutex.Lock()
		if len(chatClients) >= chatMaxConns {
			logger.Warn("Maximum chat connections reached, rejecting %s", conn.RemoteAddr())
			conn.Write([]byte("Chat room is full. Please try again later.\n"))
			conn.Close()
			chatMutex.Unlock()
			continue
		}
		chatMutex.Unlock()

		logger.Info("New chat connection from: %s", conn.RemoteAddr())
		go handleChatClient(conn)
	}
}

func handleChatClient(conn net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("Panic in handleChatClient: %v", r)
		}
		if err := conn.Close(); err != nil {
			logger.Error("Error closing chat connection: %v", err)
		}
	}()

	// Welcome message and nickname prompt
	if _, err := conn.Write([]byte(fmt.Sprintf("Welcome to %s!\nPlease enter your nickname: ", chatRoomName))); err != nil {
		logger.Error("Failed to send welcome message: %v", err)
		return
	}

	// Read nickname
	reader := bufio.NewReader(conn)
	nickname, err := reader.ReadString('\n')
	if err != nil {
		logger.Debug("Failed to read nickname from %s: %v", conn.RemoteAddr(), err)
		return
	}
	nickname = strings.TrimSpace(nickname)
	if nickname == "" {
		nickname = fmt.Sprintf("Guest-%d", time.Now().Unix()%10000)
	}

	// Check if nickname is already taken and register client atomically
	chatMutex.Lock()
	originalNick := nickname
	counter := 1
	for {
		taken := false
		for _, client := range chatClients {
			if client.Nickname == nickname {
				taken = true
				break
			}
		}
		if !taken {
			break
		}
		counter++
		nickname = fmt.Sprintf("%s%d", originalNick, counter)
	}

	clientID := fmt.Sprintf("%s-%d", conn.RemoteAddr().String(), time.Now().Unix())
	client := &ChatClient{
		Conn:     conn,
		Nickname: nickname,
		JoinTime: time.Now(),
	}
	chatClients[clientID] = client
	chatMutex.Unlock()

	// Notify about successful join
	if _, err := conn.Write([]byte(fmt.Sprintf("You joined as '%s'. Type /help for commands.\n", nickname))); err != nil {
		logger.Error("Failed to send join confirmation: %v", err)
		// Remove client from map since we couldn't confirm join
		chatMutex.Lock()
		delete(chatClients, clientID)
		chatMutex.Unlock()
		return
	}
	broadcastMessage(fmt.Sprintf("*** %s joined the chat ***", nickname), "")

	logger.Info("Chat user '%s' joined from %s", nickname, conn.RemoteAddr())

	defer func() {
		chatMutex.Lock()
		delete(chatClients, clientID)
		chatMutex.Unlock()
		broadcastMessage(fmt.Sprintf("*** %s left the chat ***", nickname), "")
		logger.Info("Chat user '%s' left", nickname)
	}()

	// Handle messages
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			logger.Debug("Chat client %s disconnected: %v", nickname, err)
			return
		}

		message = strings.TrimSpace(message)
		if message == "" {
			continue
		}

		// Handle commands
		if strings.HasPrefix(message, "/") {
			handleChatCommand(client, message)
			continue
		}

		// Broadcast regular message
		formattedMsg := fmt.Sprintf("[%s] %s: %s", time.Now().Format("15:04"), nickname, message)
		broadcastMessage(formattedMsg, clientID)
		logger.Debug("Chat message from %s: %s", nickname, message)
	}
}

func handleChatCommand(client *ChatClient, command string) {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return
	}

	cmd := strings.ToLower(parts[0])
	switch cmd {
	case "/help":
		help := `Available commands:
/help - Show this help
/list - List online users
/time - Show current time
/quit - Leave the chat
`
		client.Conn.Write([]byte(help))

	case "/list":
		chatMutex.RLock()
		userList := fmt.Sprintf("Online users (%d):\n", len(chatClients))
		for _, c := range chatClients {
			duration := time.Since(c.JoinTime).Truncate(time.Second)
			userList += fmt.Sprintf("  %s (online for %s)\n", c.Nickname, duration)
		}
		chatMutex.RUnlock()
		client.Conn.Write([]byte(userList))

	case "/time":
		timeStr := fmt.Sprintf("Current time: %s\n", time.Now().Format("2006-01-02 15:04:05"))
		client.Conn.Write([]byte(timeStr))

	case "/quit":
		client.Conn.Write([]byte("Goodbye!\n"))
		client.Conn.Close()

	default:
		client.Conn.Write([]byte(fmt.Sprintf("Unknown command: %s. Type /help for available commands.\n", cmd)))
	}
}

func broadcastMessage(message string, excludeClientID string) {
	chatMutex.RLock()
	// Create a copy of clients to avoid holding lock during network operations
	clients := make(map[string]*ChatClient)
	for id, client := range chatClients {
		if id != excludeClientID {
			clients[id] = client
		}
	}
	chatMutex.RUnlock()

	// Send messages without holding the lock
	for clientID, client := range clients {
		if _, err := client.Conn.Write([]byte(message + "\n")); err != nil {
			logger.Warn("Failed to send message to client %s: %v", clientID, err)
			// Remove failed client from the map
			chatMutex.Lock()
			if _, exists := chatClients[clientID]; exists {
				delete(chatClients, clientID)
				if closeErr := client.Conn.Close(); closeErr != nil {
					logger.Error("Error closing failed client connection: %v", closeErr)
				}
			}
			chatMutex.Unlock()
		}
	}
}
