package ws

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type WebSocketManager struct {
	Clients map[string]*websocket.Conn
	Lock    sync.RWMutex
}

func NewWebSocketManager() *WebSocketManager {
	return &WebSocketManager{
		Clients: make(map[string]*websocket.Conn),
		Lock:    sync.RWMutex{},
	}
}

func (m *WebSocketManager) ServeWebSocketRequests(c *gin.Context) {
	connID := c.Param("connID")

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	m.Lock.Lock()
	m.Clients[connID] = conn
	m.Lock.Unlock()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			m.RemoveClient(connID)
			break
		}
	}
}

func (m *WebSocketManager) SendMessage(connID string, message map[string]any) {
	m.Lock.RLock()
	conn, exists := m.Clients[connID]
	m.Lock.RUnlock()

	if !exists {
		return
	}

	if err := conn.WriteJSON(message); err != nil {
		m.RemoveClient(connID)
	}
}

func (w *WebSocketManager) RemoveClient(connID string) {
	w.Lock.Lock()
	defer w.Lock.Unlock()

	if conn, exists := w.Clients[connID]; exists {
		conn.Close()
		delete(w.Clients, connID)
	}
}

func (m *WebSocketManager) PerformPingCleanup() {
	// Implement
}
