package bot

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type LogMessage struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"` // "INFO", "WARN", "ERROR"
	Message   string    `json:"message"`
}

type LogHub struct {
	mu         sync.RWMutex
	clients    map[*websocket.Conn]bool
	backlog    []LogMessage
	maxBacklog int
}

func NewLogHub() *LogHub {
	return &LogHub{
		clients:    make(map[*websocket.Conn]bool),
		backlog:    make([]LogMessage, 0),
		maxBacklog: 100,
	}
}

func (h *LogHub) Publish(level string, message string) {
	msg := LogMessage{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
	}

	// Print to native standard log
	log.Printf("[%s] %s", level, message)

	h.mu.Lock()
	defer h.mu.Unlock()

	// Append to backlog
	h.backlog = append(h.backlog, msg)
	if len(h.backlog) > h.maxBacklog {
		h.backlog = h.backlog[1:]
	}

	// Broadcast to all active clients
	for client := range h.clients {
		err := client.WriteJSON(msg)
		if err != nil {
			client.Close()
			delete(h.clients, client)
		}
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow cross-origin connections from dev server
	},
}

func (s *BotServer) handleWebSocketLogs(w http.ResponseWriter, r *http.Request) {
	// Authentication via query parameter
	token := r.URL.Query().Get("token")
	if token != s.cfg.AdminPassword {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Websocket upgrade error: %v", err)
		return
	}

	s.logHub.mu.Lock()
	s.logHub.clients[conn] = true

	// Send backlog logs to the newly connected client immediately
	for _, msg := range s.logHub.backlog {
		_ = conn.WriteJSON(msg)
	}
	s.logHub.mu.Unlock()

	// Keep connection alive and clean up when client disconnects
	go func() {
		defer func() {
			s.logHub.mu.Lock()
			delete(s.logHub.clients, conn)
			s.logHub.mu.Unlock()
			conn.Close()
		}()

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
}

// LogInfo streams an info level log message
func (s *BotServer) LogInfo(format string, v ...interface{}) {
	s.logHub.Publish("INFO", fmt.Sprintf(format, v...))
}

// LogWarn streams a warning level log message
func (s *BotServer) LogWarn(format string, v ...interface{}) {
	s.logHub.Publish("WARN", fmt.Sprintf(format, v...))
}

// LogError streams an error level log message
func (s *BotServer) LogError(format string, v ...interface{}) {
	s.logHub.Publish("ERROR", fmt.Sprintf(format, v...))
}
