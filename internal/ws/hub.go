// Package ws provides the WebSocket hub that fans out job and repository
// events to connected browser clients.
package ws

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/KhaledSaeed18/repo-scout/internal/models"
)

// Event is a message broadcast to every connected client.
type Event struct {
	Type string         `json:"type"`
	Time int64          `json:"time"`
	Data map[string]any `json:"data"`
}

// Hub tracks clients and broadcasts events to all of them.
type Hub struct {
	mu      sync.Mutex
	clients map[*client]bool
}

type client struct {
	conn *websocket.Conn
	send chan []byte
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// New builds a Hub.
func New() *Hub {
	return &Hub{clients: map[*client]bool{}}
}

// HandleUpgrade upgrades the HTTP connection to WebSocket and registers it.
func (h *Hub) HandleUpgrade(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	c := &client{conn: conn, send: make(chan []byte, 64)}
	h.mu.Lock()
	h.clients[c] = true
	h.mu.Unlock()
	go h.writeLoop(c)
	go h.readLoop(c)
}

// Broadcast sends an event to all connected clients.
func (h *Hub) Broadcast(ev Event) {
	if ev.Time == 0 {
		ev.Time = time.Now().UnixMilli()
	}
	payload, err := json.Marshal(ev)
	if err != nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		select {
		case c.send <- payload:
		default:
			// Drop the event if the client is too slow; the frontend
			// reconciles via query refetch on the next event.
		}
	}
}

func (h *Hub) readLoop(c *client) {
	defer func() {
		h.remove(c)
		c.conn.Close()
	}()
	c.conn.SetReadLimit(1024)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
		// Clients are read-only; any inbound message just resets the deadline.
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	}
}

func (h *Hub) writeLoop(c *client) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				return
			}
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (h *Hub) remove(c *client) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
	close(c.send)
}

// JobChanged implements jobs.EventSink.
func (h *Hub) JobChanged(job *models.Job) {
	evType := "job.progress"
	if job.Status == models.JobQueued || job.Status == models.JobRunning ||
		job.Status == models.JobPaused || job.Status == models.JobCancelled ||
		job.Status == models.JobCompleted || job.Status == models.JobFailed {
		evType = "job.state_changed"
	}
	h.Broadcast(Event{Type: evType, Data: map[string]any{"job": job}})
}

// RepoChanged implements jobs.EventSink.
func (h *Hub) RepoChanged(repo *models.Repository) {
	evType := "repository.updated"
	if repo.Status == models.RepoReady {
		evType = "repository.completed"
	}
	h.Broadcast(Event{Type: evType, Data: map[string]any{"repository": repo}})
}
