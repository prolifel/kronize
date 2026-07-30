package scheduler

import (
	"database/sql"
	"log"
	"os"
	"sync"
	"time"

	"kronize/internal/db"
	"kronize/internal/model"
	"kronize/internal/runner"

	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	db      *sql.DB
	cron    *cron.Cron
	runner  *runner.Runner
	entries map[int64]cron.EntryID
	mu      sync.RWMutex
}

func cronTZ() *time.Location {
	tz := os.Getenv("TZ")
	if tz == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		log.Printf("invalid TZ %q, falling back to UTC: %v", tz, err)
		return time.UTC
	}
	return loc
}

func New(database *sql.DB, scriptsDir string) *Scheduler {
	loc := cronTZ()
	return &Scheduler{
		db:      database,
		cron:    cron.New(cron.WithLocation(loc), cron.WithParser(cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor))),
		runner:  runner.New(database, scriptsDir),
		entries: make(map[int64]cron.EntryID),
	}
}

func (s *Scheduler) Start() {
	s.cron.Start()
	go s.cleanupLoop()
}

func (s *Scheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
}

func (s *Scheduler) AddJob(job *model.Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !job.Enabled {
		return
	}
	if eid, ok := s.entries[job.ID]; ok {
		s.cron.Remove(eid)
	}
	eid, err := s.cron.AddFunc(job.CronExpression, func() {
		s.runner.ExecuteJob(job)
	})
	if err != nil {
		log.Printf("failed to schedule job %d (%s): %v", job.ID, job.Name, err)
		return
	}
	s.entries[job.ID] = eid
	log.Printf("scheduled job %d (%s): %s", job.ID, job.Name, job.CronExpression)
}

func (s *Scheduler) UpdateJob(job *model.Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if eid, ok := s.entries[job.ID]; ok {
		s.cron.Remove(eid)
		delete(s.entries, job.ID)
	}
	if job.Enabled {
		eid, err := s.cron.AddFunc(job.CronExpression, func() {
			s.runner.ExecuteJob(job)
		})
		if err != nil {
			log.Printf("failed to update job %d (%s): %v", job.ID, job.Name, err)
			return
		}
		s.entries[job.ID] = eid
	}
}

func (s *Scheduler) RemoveJob(jobID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if eid, ok := s.entries[jobID]; ok {
		s.cron.Remove(eid)
		delete(s.entries, jobID)
	}
}

func (s *Scheduler) TriggerNow(job *model.Job) {
	go s.runner.ExecuteJob(job)
}

func (s *Scheduler) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		n, err := db.DeleteOldExecutions(s.db, 30)
		if err != nil {
			log.Printf("cleanup error: %v", err)
		} else if n > 0 {
			log.Printf("cleaned up %d old executions", n)
		}
	}
}
