package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

// -----------------------------
// CONFIGURATION
// -----------------------------
const defaultPort = "8989"

var maxClients = 10

// -----------------------------
// GLOBALS
// -----------------------------
var (
	clients  = make(map[net.Conn]string)
	messages []string
	mutex    sync.Mutex
)

// -----------------------------
// ANSI COLOR CODES
// -----------------------------
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
)

// -----------------------------
// MAIN
// -----------------------------
func main() {
	port := parsePortArg()
	startServer(port)
}

// -----------------------------
// PARSE ARGUMENTS
// -----------------------------
func parsePortArg() string {
	if len(os.Args) > 2 {
		fmt.Println("[USAGE]: ./TCPChat $port")
		os.Exit(0)
	}

	port := defaultPort
	if len(os.Args) == 2 {
		port = os.Args[1]
	}
	return port
}

// -----------------------------
// SERVER START
// -----------------------------
func startServer(port string) {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer listener.Close()
	fmt.Println("Listening on the port :" + port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error:", err)
			continue
		}

		mutex.Lock()
		if len(clients) >= maxClients {
			conn.Write([]byte("Server full. Try again later.\n"))
			conn.Close()
			mutex.Unlock()
			continue
		}
		mutex.Unlock()

		go handleConnection(conn)
	}
}

// -----------------------------
// HANDLE CLIENT CONNECTION
// -----------------------------
func handleConnection(conn net.Conn) {
	defer conn.Close()

	// Send logo
	conn.Write([]byte(loadLogo()))

	// Get client name
	name := getClientName(conn)
	if name == "" {
		return
	}

	// Add client and send old messages in red
	mutex.Lock()
	clients[conn] = name
	for _, msg := range messages {
		conn.Write([]byte(ColorRed + msg + ColorReset + "\n"))
	}
	mutex.Unlock()

	// Announce join (yellow) to others only
	announce(fmt.Sprintf("%s has joined our chat...", name), conn)

	// Listen for messages
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}
		msg := formatMessage(name, text)
		mutex.Lock()
		messages = append(messages, msg)
		mutex.Unlock()
		broadcast(msg, conn)
	}

	// Client disconnect
	mutex.Lock()
	delete(clients, conn)
	mutex.Unlock()
	announce(fmt.Sprintf("%s has left our chat...", name), nil)
}

// -----------------------------
// LOAD LOGO
// -----------------------------
func loadLogo() string {
	data, err := os.ReadFile("linuxlogo.txt")
	if err != nil {
		return "Welcome to TCP-Chat!\n[ENTER YOUR NAME]: "
	}
	return string(data) + "\n"
}

// -----------------------------
// GET CLIENT NAME (unique)
// -----------------------------
func getClientName(conn net.Conn) string {
	scanner := bufio.NewScanner(conn)
	conn.Write([]byte("\n[ENTER YOUR NAME]: "))
	for {
		if !scanner.Scan() {
			return ""
		}
		name := strings.TrimSpace(scanner.Text())
		if name == "" {
			conn.Write([]byte("\n[ENTER YOUR NAME]: "))
			continue
		}

		mutex.Lock()
		nameTaken := false
		for _, n := range clients {
			if n == name {
				nameTaken = true
				break
			}
		}
		mutex.Unlock()

		if nameTaken {
			conn.Write([]byte("Name already taken. Choose another name:\n[ENTER YOUR NAME]: "))
			continue
		}

		return name
	}
}

// -----------------------------
// BROADCAST
// -----------------------------
func broadcast(msg string, sender net.Conn) {
	mutex.Lock()
	defer mutex.Unlock()
	for c := range clients {
		switch {
		case c == sender:
			// Current user sees full message with timestamp and username in green
			c.Write([]byte(ColorGreen + msg + ColorReset + "\n"))
		default:
			// Others see full message in blue
			c.Write([]byte(ColorBlue + msg + ColorReset + "\n"))
		}
	}
}

// -----------------------------
// ANNOUNCE SYSTEM
// -----------------------------
func announce(msg string, excludeConn net.Conn) {
	mutex.Lock()
	messages = append(messages, msg)
	for c := range clients {
		if c != excludeConn {
			c.Write([]byte(ColorYellow + msg + ColorReset + "\n"))
		}
	}
	mutex.Unlock()
}

// -----------------------------
// FORMAT MESSAGE
// -----------------------------
func formatMessage(name, text string) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	return fmt.Sprintf("[%s][%s]:%s", timestamp, name, text)
}
