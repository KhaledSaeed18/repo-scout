// Package jobs provides a persistent background job queue with a worker pool
// that supports pause, resume, and cancel. Job state lives in SQLite so work
// survives process restarts.
package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/KhaledSaeed18/repo-scout/internal/config"
	"github.com/KhaledSaeed18/repo-scout/internal/models"
)

// Reporter tracks progress for an in-flight job. The runner calls these
// methods from its stage implementations.
type Reporter interface {
	// SetTotal declares the total work units.
	SetTotal(n int)
	// SetProgress sets the fraction of work completed, in [0, 1].
	SetProgress(frac float64)
	// Inc adds completed work units.
	Inc(n int)
	// SetMessage updates the human-readable status message.
	SetMessage(msg string)
	// Checkpoint blocks while the job is paused and reports cancellation.
	Checkpoint(ctx context.Context) error
}

// Runner executes a job. It receives the repository id, a progress reporter,
// and the effective settings.
type Runner interface {
	Run(ctx context.Context, repoID, jobID uint, rep Reporter, settings config.Settings) error
}

// EventSink receives job lifecycle events for broadcast to WebSocket clients.
type EventSink interface {
	// JobChanged is called whenever a job's state or progress changes.
	JobChanged(job *models.Job)
	// RepoChanged is called when repository status or summary changes.
	RepoChanged(repo *models.Repository)
}

// Manager owns the job queue and worker pool.
type Manager struct {
	db        *gorm.DB
	runner    Runner
	settings  func() config.Settings
	eventSink EventSink
	log       *slog.Logger

	mu     sync.Mutex
	active map[uint]*activeJob
	wake   chan struct{}

	baseCtx context.Context
}

// activeJob tracks an in-flight job for pause/cancel signaling.
type activeJob struct {
	cancel context.CancelFunc
	mu     sync.Mutex
	paused bool
	notify chan struct{}
}

// New builds a Manager. settings, eventSink and logger may be nil: defaults
// are used, events are dropped and nothing is logged.
func New(db *gorm.DB, runner Runner, settings func() config.Settings, sink EventSink, logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	return &Manager{
		db:        db,
		runner:    runner,
		settings:  settings,
		eventSink: sink,
		log:       logger,
		active:    map[uint]*activeJob{},
		wake:      make(chan struct{}, 1),
	}
}

// Start recovers interrupted jobs and launches the worker pool. It blocks
// until ctx is cancelled.
func (m *Manager) Start(ctx context.Context) error {
	if err := m.recover(); err != nil {
		return fmt.Errorf("recover jobs: %w", err)
	}
	n := 1
	if m.settings != nil {
		if c := m.settings().WorkerCount; c > 0 {
			n = c
		}
	}
	m.baseCtx = ctx
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.worker(ctx)
		}()
	}
	<-ctx.Done()
	wg.Wait()
	return nil
}

// recover marks stale in-flight jobs as interrupted and re-queues them so
// work continues after a crash. Repositories stuck in "scanning" go back to
// ready when an earlier scan's results are still in place, and are flagged
// failed when they were never scanned.
func (m *Manager) recover() error {
	now := time.Now()
	// Jobs the user was cancelling are finished, never resumed.
	err := m.db.Model(&models.Job{}).
		Where("status = ?", models.JobCancelling).
		Updates(map[string]any{"status": models.JobCancelled, "message": "cancelled", "finished_at": now, "updated_at": now}).Error
	if err != nil {
		return err
	}
	err = m.db.Model(&models.Job{}).
		Where("status IN ?", []string{models.JobRunning, models.JobPaused}).
		Updates(map[string]any{"status": models.JobInterrupted, "message": "interrupted by restart", "updated_at": now}).Error
	if err != nil {
		return err
	}
	err = m.db.Model(&models.Job{}).
		Where("status = ?", models.JobInterrupted).
		Updates(map[string]any{"status": models.JobQueued, "message": "re-queued after restart", "updated_at": now}).Error
	if err != nil {
		return err
	}
	err = m.db.Model(&models.Repository{}).
		Where("status = ? AND last_scanned_at IS NOT NULL", models.RepoScanning).
		Updates(map[string]any{"status": models.RepoReady, "updated_at": now}).Error
	if err != nil {
		return err
	}
	return m.db.Model(&models.Repository{}).
		Where("status = ?", models.RepoScanning).
		Updates(map[string]any{"status": models.RepoFailed, "updated_at": now}).Error
}

// Enqueue creates a queued job and wakes a worker.
func (m *Manager) Enqueue(repoID uint, kind string) (*models.Job, error) {
	// A repository has at most one live job of a kind; asking again returns
	// it, so two scans never clear and rewrite the same data concurrently.
	var job models.Job
	err := m.db.Transaction(func(tx *gorm.DB) error {
		live := []string{models.JobQueued, models.JobRunning, models.JobPaused, models.JobCancelling}
		err := tx.Where("repo_id = ? AND kind = ? AND status IN ?", repoID, kind, live).Order("id DESC").First(&job).Error
		if err == nil {
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		job = models.Job{RepoID: repoID, Kind: kind, Status: models.JobQueued}
		return tx.Create(&job).Error
	})
	if err != nil {
		return nil, fmt.Errorf("enqueue job: %w", err)
	}
	m.broadcast(&job)
	select {
	case m.wake <- struct{}{}:
	default:
	}
	return &job, nil
}

// List returns jobs, newest first.
func (m *Manager) List(limit int) ([]models.Job, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var out []models.Job
	if err := m.db.Order("id DESC").Limit(limit).Find(&out).Error; err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	return out, nil
}

// Pause requests that a running job pause at its next checkpoint.
func (m *Manager) Pause(jobID uint) error {
	aj := m.lookup(jobID)
	if aj == nil {
		return fmt.Errorf("job %d is not running", jobID)
	}
	// Only a running job can pause; the guard loses gracefully to a job that
	// finished in the meantime instead of leaving it "paused" with no worker.
	if err := m.transition(jobID, []string{models.JobRunning}, models.JobPaused, "paused"); err != nil {
		return err
	}
	aj.mu.Lock()
	aj.paused = true
	aj.mu.Unlock()
	return nil
}

func (m *Manager) Resume(jobID uint) error {
	if err := m.transition(jobID, []string{models.JobPaused}, models.JobRunning, "resumed"); err != nil {
		return err
	}
	if aj := m.lookup(jobID); aj != nil {
		aj.mu.Lock()
		aj.paused = false
		close(aj.notify)
		aj.notify = make(chan struct{})
		aj.mu.Unlock()
	}
	return nil
}

func (m *Manager) Cancel(jobID uint) error {
	// A queued job has no worker yet, so it can be finished directly. The
	// status guard keeps this safe if a worker claims it at the same time.
	now := time.Now()
	res := m.db.Model(&models.Job{}).
		Where("id = ? AND status = ?", jobID, models.JobQueued).
		Updates(map[string]any{"status": models.JobCancelled, "message": "cancelled", "finished_at": now, "updated_at": now})
	if res.Error != nil {
		return fmt.Errorf("cancel job %d: %w", jobID, res.Error)
	}
	if res.RowsAffected == 1 {
		var job models.Job
		if err := m.db.First(&job, jobID).Error; err == nil {
			m.broadcast(&job)
		}
		return nil
	}

	aj := m.lookup(jobID)
	if aj == nil {
		var job models.Job
		if err := m.db.First(&job, jobID).Error; err != nil {
			return err
		}
		return fmt.Errorf("job %d is %s and cannot be cancelled", jobID, job.Status)
	}
	// Record the intent before signalling, and only while the job is live:
	// the worker's final write must always be the last one.
	if err := m.transition(jobID, []string{models.JobRunning, models.JobPaused}, models.JobCancelling, "cancelling"); err != nil {
		return err
	}
	aj.mu.Lock()
	aj.paused = false
	close(aj.notify)
	aj.notify = make(chan struct{})
	aj.mu.Unlock()
	aj.cancel()
	return nil
}

// transition moves a job to status only if it is currently in one of from,
// and broadcasts the change.
func (m *Manager) transition(jobID uint, from []string, status, msg string) error {
	res := m.db.Model(&models.Job{}).
		Where("id = ? AND status IN ?", jobID, from).
		Updates(map[string]any{"status": status, "message": msg, "updated_at": time.Now()})
	if res.Error != nil {
		return fmt.Errorf("update job %d: %w", jobID, res.Error)
	}
	var job models.Job
	if err := m.db.First(&job, jobID).Error; err != nil {
		return err
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("job %d is %s", jobID, job.Status)
	}
	m.broadcast(&job)
	return nil
}

func (m *Manager) lookup(jobID uint) *activeJob {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.active[jobID]
}

func (m *Manager) worker(ctx context.Context) {
	for {
		job := m.claimNext(ctx)
		if job == nil {
			select {
			case <-ctx.Done():
				return
			case <-m.wake:
			}
			continue
		}
		m.run(job)
	}
}

// claimNext atomically claims the oldest queued job.
func (m *Manager) claimNext(ctx context.Context) *models.Job {
	var job models.Job
	err := m.db.Transaction(func(tx *gorm.DB) error {
		var candidate models.Job
		if err := tx.Where("status = ?", models.JobQueued).Order("id ASC").First(&candidate).Error; err != nil {
			return err
		}
		if err := tx.Model(&candidate).Updates(map[string]any{
			"status": models.JobRunning, "started_at": time.Now(), "message": "running", "updated_at": time.Now(),
		}).Error; err != nil {
			return err
		}
		job = candidate
		return nil
	})
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) && ctx.Err() == nil {
			m.log.Error("claim job", "err", err)
		}
		return nil
	}
	return &job
}

func (m *Manager) run(job *models.Job) {
	jctx, cancel := context.WithCancel(m.baseCtx)
	aj := &activeJob{cancel: cancel, notify: make(chan struct{})}
	m.mu.Lock()
	m.active[job.ID] = aj
	m.mu.Unlock()
	defer func() {
		m.mu.Lock()
		delete(m.active, job.ID)
		m.mu.Unlock()
	}()

	rep := &jobReporter{m: m, job: job, aj: aj}
	settings := config.Defaults()
	if m.settings != nil {
		settings = m.settings()
	}

	log := m.log.With("job", job.ID, "repo", job.RepoID, "kind", job.Kind)
	log.Info("job started")
	started := time.Now()
	err := m.runner.Run(jctx, job.RepoID, job.ID, rep, settings)
	took := time.Since(started).Round(time.Millisecond).String()

	var cur models.Job
	m.db.First(&cur, job.ID)
	now := time.Now()
	final := map[string]any{"updated_at": now, "finished_at": now}
	switch {
	case err == nil:
		final["status"] = models.JobCompleted
		final["message"] = "completed"
		final["progress"] = 1
		log.Info("job completed", "took", took)
	case cur.Status == models.JobCancelling || errors.Is(err, context.Canceled):
		final["status"] = models.JobCancelled
		final["message"] = "cancelled"
		final["error"] = ""
		log.Info("job cancelled", "took", took)
	default:
		final["status"] = models.JobFailed
		final["message"] = "failed"
		final["error"] = err.Error()
		log.Error("job failed", "took", took, "err", err)
	}
	if err := m.db.Model(&models.Job{}).Where("id = ?", job.ID).Updates(final).Error; err != nil {
		log.Error("record job result", "err", err)
	}
	m.db.First(&cur, job.ID)
	m.broadcast(&cur)

	// The scan rewrote the repository's data; tell clients to refresh it.
	var repo models.Repository
	if m.eventSink != nil && m.db.First(&repo, job.RepoID).Error == nil {
		m.eventSink.RepoChanged(&repo)
	}
}

func (m *Manager) broadcast(job *models.Job) {
	if m.eventSink != nil {
		m.eventSink.JobChanged(job)
	}
}

// jobReporter persists progress to the database in a throttled manner.
type jobReporter struct {
	m   *Manager
	job *models.Job
	aj  *activeJob

	mu        sync.Mutex
	current   int
	total     int
	message   string
	progress  float64
	lastWrite time.Time
}

func (r *jobReporter) SetTotal(n int) {
	r.mu.Lock()
	r.total = n
	r.mu.Unlock()
}

func (r *jobReporter) SetProgress(frac float64) {
	r.mu.Lock()
	r.progress = frac
	r.mu.Unlock()
	r.flush(false)
}

func (r *jobReporter) Inc(n int) {
	r.mu.Lock()
	r.current += n
	r.mu.Unlock()
	r.flush(false)
}

func (r *jobReporter) SetMessage(msg string) {
	r.mu.Lock()
	r.message = msg
	r.mu.Unlock()
	r.flush(true)
}

func (r *jobReporter) Checkpoint(ctx context.Context) error {
	r.aj.mu.Lock()
	paused := r.aj.paused
	notify := r.aj.notify
	r.aj.mu.Unlock()
	if !paused {
		return nil
	}
	select {
	case <-notify:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *jobReporter) flush(force bool) {
	r.mu.Lock()
	now := time.Now()
	if !force && now.Sub(r.lastWrite) < 150*time.Millisecond {
		r.mu.Unlock()
		return
	}
	frac := r.progress
	if r.total > 0 && frac == 0 {
		frac = float64(r.current) / float64(r.total)
	}
	cur := r.current
	msg := r.message
	last := r.lastWrite
	r.lastWrite = now
	r.mu.Unlock()

	if last.IsZero() && !force {
		return
	}
	updates := map[string]any{"progress": frac, "current": cur, "message": msg, "updated_at": now}
	if err := r.m.db.Model(&models.Job{}).Where("id = ?", r.job.ID).Updates(updates).Error; err != nil {
		return
	}
	var job models.Job
	if err := r.m.db.First(&job, r.job.ID).Error; err == nil {
		r.m.broadcast(&job)
	}
}
