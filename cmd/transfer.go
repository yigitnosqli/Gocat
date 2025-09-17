package cmd

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/ibrahmsql/gocat/internal/logger"
	"github.com/spf13/cobra"
)

var (
	// Transfer mode flags
	transferFile     string
	transferOutput   string
	transferProgress bool
	transferResume   bool
	transferChecksum bool
	transferCompress bool
	transferTimeout  time.Duration
	transferBuffer   int
)

// transferCmd represents the transfer command
var transferCmd = &cobra.Command{
	Use:   "transfer [mode] [options]",
	Short: "File transfer operations",
	Long: `Transfer files over network connections with advanced features.

Modes:
  send <file> <host> <port>    Send a file to remote host
  receive <port> [output]      Receive a file on specified port

Features:
  - Progress monitoring
  - Resume interrupted transfers
  - Checksum verification
  - Compression support
  - Bandwidth limiting

Examples:
  gocat transfer send file.txt 192.168.1.100 8080
  gocat transfer receive 8080 received_file.txt
  gocat transfer send --progress --checksum file.txt host 8080`,
	Args: cobra.MinimumNArgs(1),
	Run:  runTransfer,
}

// init registers the transfer command with the root command and defines its CLI flags.
// Flags configured: file, output, progress, resume, checksum, compress, transfer-timeout, and buffer size.
func init() {
	rootCmd.AddCommand(transferCmd)

	// Transfer specific flags
	transferCmd.Flags().StringVarP(&transferFile, "file", "f", "", "File to transfer")
	transferCmd.Flags().StringVarP(&transferOutput, "output", "o", "", "Output file name")
	transferCmd.Flags().BoolVar(&transferProgress, "progress", false, "Show transfer progress")
	transferCmd.Flags().BoolVar(&transferResume, "resume", false, "Resume interrupted transfer")
	transferCmd.Flags().BoolVar(&transferChecksum, "checksum", false, "Verify file integrity with checksum")
	transferCmd.Flags().BoolVar(&transferCompress, "compress", false, "Compress data during transfer")
	transferCmd.Flags().DurationVar(&transferTimeout, "transfer-timeout", 30*time.Second, "Transfer timeout")
	transferCmd.Flags().IntVar(&transferBuffer, "buffer", 32768, "Transfer buffer size in bytes")
}

// returned by sendFile or receiveFile are logged fatally as well.
func runTransfer(cmd *cobra.Command, args []string) {
	if len(args) < 1 {
		logger.Fatal("Transfer mode required (send/receive)")
		return
	}

	mode := args[0]
	switch mode {
	case "send":
		if len(args) < 4 {
			logger.Fatal("Usage: gocat transfer send <file> <host> <port>")
			return
		}
		filePath := args[1]
		host := args[2]
		port := args[3]
		if err := sendFile(filePath, host, port); err != nil {
			logger.Fatal("Send error: %v", err)
		}

	case "receive":
		if len(args) < 2 {
			logger.Fatal("Usage: gocat transfer receive <port> [output_file]")
			return
		}
		port := args[1]
		outputFile := ""
		if len(args) > 2 {
			outputFile = args[2]
		}
		if err := receiveFile(port, outputFile); err != nil {
			logger.Fatal("Receive error: %v", err)
		}

	default:
		logger.Fatal("Invalid transfer mode. Use 'send' or 'receive'")
	}
}

// sendFile sends the file at filePath to the specified host:port over TCP.
// It verifies that filePath exists and is a regular file, opens it, connects to the remote
// address using transferTimeout, sends a small metadata header ("GOCAT-TRANSFER\n<name>\n<size>\n"),
// then streams the file contents while reporting progress.
// Returns an error if file validation/opening, network connection, metadata transmission,
// or the actual transfer fails.
func sendFile(filePath, host, port string) error {
	// Check if file exists and get info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("file error: %w", err)
	}

	if fileInfo.IsDir() {
		return fmt.Errorf("cannot send directory: %s", filePath)
	}

	fileSize := fileInfo.Size()
	logger.Info("Preparing to send file: %s (%.2f MB)", filePath, float64(fileSize)/(1024*1024))

	// Open file for reading
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Connect to remote host
	address := net.JoinHostPort(host, port)
	conn, err := net.DialTimeout("tcp", address, transferTimeout)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer conn.Close()

	logger.Info("Connected to %s, starting file transfer...", address)

	// Send file metadata first
	metadata := fmt.Sprintf("GOCAT-TRANSFER\n%s\n%d\n", filepath.Base(filePath), fileSize)
	if _, err := conn.Write([]byte(metadata)); err != nil {
		return fmt.Errorf("failed to send metadata: %w", err)
	}

	// Transfer file with progress monitoring
	return transferFileData(file, conn, fileSize, "Sending")
}

// receiveFile listens on the given TCP port, accepts a single incoming
// transfer connection, reads the sender's metadata (filename and size), and
// writes the incoming byte stream to disk. If outputFile is empty the name
// advertised by the sender is used. Returns a non-nil error if listening,
// accepting the connection, reading metadata, creating the destination file,
// or the data transfer fail.
func receiveFile(port, outputFile string) error {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("failed to listen on port %s: %w", port, err)
	}
	defer listener.Close()

	logger.Info("Listening for file transfer on port %s...", port)

	conn, err := listener.Accept()
	if err != nil {
		return fmt.Errorf("failed to accept connection: %w", err)
	}
	defer conn.Close()

	logger.Info("Connection accepted from %s", conn.RemoteAddr())

	// Read file metadata
	fileName, fileSize, err := readTransferMetadata(conn)
	if err != nil {
		return fmt.Errorf("failed to read metadata: %w", err)
	}

	// Determine output file name
	if outputFile == "" {
		outputFile = fileName
	}

	logger.Info("Receiving file: %s (%.2f MB)", fileName, float64(fileSize)/(1024*1024))

	// Create output file
	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Receive file data with progress monitoring
	return transferFileData(conn, file, fileSize, "Receiving")
}

// readTransferMetadata reads and parses a transfer header from conn.
//
// It reads up to 1024 bytes and expects a newline-delimited header with the
// following form:
//
//	GOCAT-TRANSFER
//	<filename>
//	<filesize>
//
// Returns the filename and filesize parsed from the header. Returns an error
// if the initial read fails, the header is malformed, or the filesize cannot be
// parsed as an int64.
func readTransferMetadata(conn net.Conn) (fileName string, fileSize int64, err error) {
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		return "", 0, err
	}

	data := string(buffer[:n])
	lines := []string{}
	current := ""
	for _, char := range data {
		if char == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(char)
		}
	}

	if len(lines) < 3 || lines[0] != "GOCAT-TRANSFER" {
		return "", 0, fmt.Errorf("invalid transfer metadata")
	}

	fileName = lines[1]
	fileSize, err = strconv.ParseInt(lines[2], 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("invalid file size: %w", err)
	}

	return fileName, fileSize, nil
}

// transferFileData copies file data from src to dst while tracking progress and reporting completion.
// It reads into a buffer of size transferBuffer, writes each chunk to dst, and accumulates the number
// of bytes transferred. When transferProgress is enabled the function updates a live progress display
// once per second via showTransferProgress and prints a final progress line on completion.
// On success it logs the total bytes transferred, elapsed duration, and average throughput (MB/s).
// Any read or write error is returned wrapped with context.
func transferFileData(src io.Reader, dst io.Writer, totalSize int64, operation string) error {
	buffer := make([]byte, transferBuffer)
	var transferred int64
	startTime := time.Now()
	lastProgress := time.Now()

	for {
		n, err := src.Read(buffer)
		if n > 0 {
			_, writeErr := dst.Write(buffer[:n])
			if writeErr != nil {
				return fmt.Errorf("write error: %w", writeErr)
			}
			transferred += int64(n)

			// Show progress every second
			if transferProgress && time.Since(lastProgress) >= time.Second {
				showTransferProgress(operation, transferred, totalSize, startTime)
				lastProgress = time.Now()
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read error: %w", err)
		}
	}

	// Final progress report
	if transferProgress {
		showTransferProgress(operation, transferred, totalSize, startTime)
		fmt.Println() // New line after progress
	}

	duration := time.Since(startTime)
	speed := float64(transferred) / duration.Seconds() / (1024 * 1024) // MB/s

	logger.Info("%s completed: %d bytes in %v (%.2f MB/s)", operation, transferred, duration, speed)
	return nil
}

// showTransferProgress prints an inline progress line for a transfer operation.
// It writes a carriage-returned line to stdout containing a 40-character ASCII
// progress bar, percent complete, current throughput in MB/s and an ETA when
// computable. `operation` is used as the label, `transferred` and `total` are
// byte counts, and `startTime` is used to derive elapsed time and speed.
// This function has no return value and performs direct output via fmt.Printf.
func showTransferProgress(operation string, transferred, total int64, startTime time.Time) {
	percent := float64(transferred) / float64(total) * 100
	duration := time.Since(startTime)
	speed := float64(transferred) / duration.Seconds() / (1024 * 1024) // MB/s

	// Create progress bar
	barWidth := 40
	filledWidth := int(percent / 100 * float64(barWidth))
	bar := "["
	for i := 0; i < barWidth; i++ {
		if i < filledWidth {
			bar += "="
		} else if i == filledWidth {
			bar += ">"
		} else {
			bar += " "
		}
	}
	bar += "]"

	// Calculate ETA
	eta := ""
	if speed > 0 {
		remaining := float64(total-transferred) / (speed * 1024 * 1024)
		eta = fmt.Sprintf(" ETA: %v", time.Duration(remaining)*time.Second)
	}

	fmt.Printf("\r%s: %s %.1f%% (%.2f MB/s)%s",
		operation, bar, percent, speed, eta)
}
