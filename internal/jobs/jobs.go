// Package jobs provides a persistent background job queue with a worker pool
// that supports pause, resume, and cancel. Job state lives in SQLite so work
// survives process restarts.
package jobs

import (
	"context"
	"errors"
	"fmt"
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
	db          *gorm.DB
	runner      Runner
	settings    func() config.Settings
	eventSink   EventSink
	workerCount int

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

// New builds a Manager. settings and eventSink may be nil (defaults used).
func New(db *gorm.DB, runner Runner, settings func() config.Settings, sink EventSink) *Manager {
	return &Manager{
		db:          db,
		runner:      runner,
		settings:    settings,
		eventSink:   sink,
		workerCount: 0,
		active:      map[uint]*activeJob{},
		wake:        make(chan struct{}, 1),
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
	if m.workerCount > 0 {
		n = m.workerCount
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
// work continues after a crash. Repositories stuck in "scanning" are flagged
// failed so they can be rescanned.
func (m *Manager) recover() error {
	now := time.Now()
	err := m.db.Model(&models.Job{}).
		Where("status IN ?", []string{models.JobRunning, models.JobCancelling}).
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
	return m.db.Model(&models.Repository{}).
		Where("status = ?", "scanning").
		Updates(map[string]any{"status": models.RepoFailed, "updated_at": now}).Error
}

// Enqueue creates a queued job and wakes a worker.
func (m *Manager) Enqueue(repoID uint, kind string) (*models.Job, error) {
	job := models.Job{
		RepoID: repoID,
		Kind:   kind,
		Status: models.JobQueued,
	}
	if err := m.db.Create(&job).Error; err != nil {
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
	aj.mu.Lock()
	aj.paused = true
	aj.mu.Unlock()
	if err := m.updateState(jobID, models.JobPaused, "paused"); err != nil {
		return err
	}
	return nil
}

// Resume un-pauses a paused job.
func (m *Manager) Resume(jobID uint) error {
	var job models.Job
	if err := m.db.First(&job, jobID).Error; err != nil {
		return err
	}
	if job.Status != models.JobPaused {
		return fmt.Errorf("job %d is not paused", jobID)
	}
	if aj := m.lookup(jobID); aj != nil {
		aj.mu.Lock()
		aj.paused = false
		close(aj.notify)
		aj.notify = make(chan struct{})
		aj.mu.Unlock()
	}
	if err := m.updateState(jobID, models.JobRunning, "resumed"); err != nil {
		return err
	}
	return nil
}

// Cancel stops a job. Queued jobs are cancelled immediately; running and
// paused jobs are cancelled at the next checkpoint.
func (m *Manager) Cancel(jobID uint) error {
	var job models.Job
	if err := m.db.First(&job, jobID).Error; err != nil {
		return err
	}
	if aj := m.lookup(jobID); aj != nil {
		aj.mu.Lock()
		aj.paused = false
		close(aj.notify)
		aj.notify = make(chan struct{})
		aj.mu.Unlock()
		aj.cancel()
	}
	return m.updateState(jobID, models.JobCancelling, "cancelling")
}

func (m *Manager) lookup(jobID uint) *activeJob {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.active[jobID]
}

func (m *Manager) updateState(jobID uint, status, msg string) error {
	err := m.db.Model(&models.Job{}).
		Where("id = ?", jobID).
		Updates(map[string]any{"status": status, "message": msg, "updated_at": time.Now()}).Error
	if err != nil {
		return fmt.Errorf("update job %d: %w", jobID, err)
	}
	var job models.Job
	if err := m.db.First(&job, jobID).Error; err == nil {
		m.broadcast(&job)
	}
	return nil
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
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

	err := m.runner.Run(jctx, job.RepoID, job.ID, rep, settings)

	var cur models.Job
	m.db.First(&cur, job.ID)
	now := time.Now()
	final := map[string]any{"updated_at": now, "finished_at": now}
	switch {
	case err == nil:
		final["status"] = models.JobCompleted
		final["message"] = "completed"
		final["progress"] = 1
	case cur.Status == models.JobCancelling || errors.Is(err, context.Canceled):
		final["status"] = models.JobCancelled
		final["message"] = "cancelled"
		final["error"] = ""
	default:
		final["status"] = models.JobFailed
		final["message"] = "failed"
		final["error"] = err.Error()
	}
	m.db.Model(&models.Job{}).Where("id = ?", job.ID).Updates(final)
	m.db.First(&cur, job.ID)
	m.broadcast(&cur)
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
