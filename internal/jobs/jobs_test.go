package jobs

import (
	"context"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/KhaledSaeed18/repo-scout/internal/config"
	"github.com/KhaledSaeed18/repo-scout/internal/database"
	"github.com/KhaledSaeed18/repo-scout/internal/models"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// blockingRunner blocks until its context is cancelled, checkpointing first so
// pause can take effect.
type blockingRunner struct {
	mu         sync.Mutex
	started    chan struct{}
	checkpoint chan struct{}
	pausedSeen bool
}

func (b *blockingRunner) Run(ctx context.Context, repoID, jobID uint, rep Reporter, settings config.Settings) error {
	select {
	case b.started <- struct{}{}:
	default:
	}
	if err := rep.Checkpoint(ctx); err != nil {
		return err
	}
	b.mu.Lock()
	b.pausedSeen = true
	b.mu.Unlock()
	select {
	case <-b.checkpoint:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TestManagerLifecycle(t *testing.T) {
	db := testDB(t)
	br := &blockingRunner{started: make(chan struct{}, 1), checkpoint: make(chan struct{})}
	m := New(db, br, func() config.Settings { s := config.Defaults(); s.WorkerCount = 1; return s }, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Start(ctx) }()
	defer cancel()

	job, err := m.Enqueue(1, "scan")
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	<-br.started

	if err := m.Pause(job.ID); err != nil {
		t.Fatalf("pause: %v", err)
	}
	var j models.Job
	db.First(&j, job.ID)
	if j.Status != models.JobPaused {
		t.Fatalf("expected paused, got %s", j.Status)
	}

	if err := m.Resume(job.ID); err != nil {
		t.Fatalf("resume: %v", err)
	}
	br.mu.Lock()
	pausedSeen := br.pausedSeen
	br.mu.Unlock()
	if !pausedSeen {
		t.Fatalf("runner never observed pause")
	}

	if err := m.Cancel(job.ID); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	deadline := time.After(3 * time.Second)
	for {
		db.First(&j, job.ID)
		if j.Status == models.JobCancelled || j.Status == models.JobCompleted || j.Status == models.JobFailed {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("job stuck in %s", j.Status)
		case <-time.After(20 * time.Millisecond):
		}
	}
	if j.Status != models.JobCancelled {
		t.Fatalf("expected cancelled, got %s", j.Status)
	}
}

func TestManagerRecoversInterrupted(t *testing.T) {
	db := testDB(t)
	db.Create(&models.Job{RepoID: 1, Kind: "scan", Status: models.JobRunning})
	db.Create(&models.Job{RepoID: 2, Kind: "scan", Status: models.JobInterrupted})
	db.Create(&models.Repository{Name: "r", Path: "/tmp/x", Status: "scanning"})

	m := New(db, &blockingRunner{}, nil, nil)
	if err := m.recover(); err != nil {
		t.Fatalf("recover: %v", err)
	}
	var running, requeued int64
	db.Model(&models.Job{}).Where("status = ?", models.JobRunning).Count(&running)
	db.Model(&models.Job{}).Where("status = ?", models.JobQueued).Count(&requeued)
	if running != 0 || requeued != 2 {
		t.Fatalf("recovery wrong: running=%d requeued=%d", running, requeued)
	}
	var repo models.Repository
	db.First(&repo)
	if repo.Status != models.RepoFailed {
		t.Fatalf("expected repo failed after recovery, got %s", repo.Status)
	}
}

func TestManagerEnqueueCompletion(t *testing.T) {
	db := testDB(t)
	m := New(db, &doneRunner{}, func() config.Settings { s := config.Defaults(); s.WorkerCount = 1; return s }, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Start(ctx) }()
	defer cancel()

	job, err := m.Enqueue(1, "scan")
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	deadline := time.After(3 * time.Second)
	var j models.Job
	for {
		db.First(&j, job.ID)
		if j.Status == models.JobCompleted || j.Status == models.JobFailed {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("job stuck in %s", j.Status)
		case <-time.After(20 * time.Millisecond):
		}
	}
	if j.Status != models.JobCompleted {
		t.Fatalf("expected completed, got %s (%s)", j.Status, j.Error)
	}
}

type doneRunner struct{}

func (doneRunner) Run(ctx context.Context, repoID, jobID uint, rep Reporter, settings config.Settings) error {
	rep.SetTotal(10)
	rep.Inc(10)
	rep.SetProgress(1)
	return nil
}
