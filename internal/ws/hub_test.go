package ws

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/KhaledSaeed18/repo-scout/internal/models"
)

func TestBroadcastJSON(t *testing.T) {
	h := New()
	payloads := make(chan []byte, 8)
	c := &client{send: make(chan []byte, 8)}
	h.mu.Lock()
	h.clients[c] = true
	h.mu.Unlock()

	h.Broadcast(Event{Type: "job.progress", Data: map[string]any{"progress": 0.5}})

	var ev Event
	select {
	case raw := <-c.send:
		if err := json.Unmarshal(raw, &ev); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("no broadcast received")
	}
	if ev.Type != "job.progress" || ev.Time == 0 {
		t.Fatalf("unexpected event %+v", ev)
	}
	_ = payloads
}

func TestEventSink(t *testing.T) {
	h := New()
	job := &models.Job{ID: 3, Status: models.JobRunning}
	h.JobChanged(job)
	repo := &models.Repository{ID: 2, Status: models.RepoReady}
	h.RepoChanged(repo)
}
