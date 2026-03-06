// Package backup provides automated SQLite database backup with configurable
// scheduling and retention. Backups are created atomically using VACUUM INTO,
// which produces a consistent snapshot even while the database is being written.
package backup

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/standardws/operator/pkg/logger"
)

// Config holds backup configuration.
type Config struct {
	// Interval between automated backups (e.g. 1h, 6h, 24h).
	Interval time.Duration

	// BackupDir is the directory where backup files are stored.
	BackupDir string

	// MaxBackups is the maximum number of local backups to retain.
	// Older backups are pruned after a successful new backup.
	// 0 means unlimited.
	MaxBackups int

	// DBPath is the path to the SQLite database file being backed up.
	DBPath string
}

// DefaultConfig returns a sensible default configuration.
// The caller must set DBPath and BackupDir.
func DefaultConfig() Config {
	return Config{
		Interval:   6 * time.Hour,
		MaxBackups: 7,
	}
}

// Scheduler runs automated backups on a configurable interval.
type Scheduler struct {
	cfg    Config
	db     *sql.DB
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.Mutex

	// lastBackup records the time of the most recent successful backup.
	lastBackup time.Time
	// lastError records the error from the most recent backup attempt.
	lastError error
}

// NewScheduler creates a backup scheduler. The provided *sql.DB must point
// to the same database file specified in cfg.DBPath (it is used only for
// VACUUM INTO; the scheduler never modifies data).
func NewScheduler(db *sql.DB, cfg Config) (*Scheduler, error) {
	if cfg.BackupDir == "" {
		return nil, fmt.Errorf("backup: BackupDir is required")
	}
	if cfg.DBPath == "" {
		return nil, fmt.Errorf("backup: DBPath is required")
	}
	if cfg.Interval <= 0 {
		return nil, fmt.Errorf("backup: Interval must be positive")
	}

	// Ensure backup directory exists.
	if err := os.MkdirAll(cfg.BackupDir, 0o755); err != nil {
		return nil, fmt.Errorf("backup: create dir %q: %w", cfg.BackupDir, err)
	}

	return &Scheduler{
		cfg: cfg,
		db:  db,
	}, nil
}

// Start begins the periodic backup loop. It is safe to call Start only once.
func (s *Scheduler) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.loop(ctx)
	}()

	logger.Info(fmt.Sprintf("Backup scheduler started (interval=%s, dir=%s, maxBackups=%d)",
		s.cfg.Interval, s.cfg.BackupDir, s.cfg.MaxBackups))
}

// Stop gracefully shuts down the scheduler and waits for any in-progress
// backup to finish.
func (s *Scheduler) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
	logger.Info("Backup scheduler stopped")
}

// RunOnce performs a single backup immediately. Safe for manual/test use.
func (s *Scheduler) RunOnce() error {
	return s.doBackup()
}

// LastBackup returns the time of the most recent successful backup.
func (s *Scheduler) LastBackup() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastBackup
}

// LastError returns the error from the most recent backup attempt (nil if successful).
func (s *Scheduler) LastError() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastError
}

func (s *Scheduler) loop(ctx context.Context) {
	ticker := time.NewTicker(s.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.doBackup(); err != nil {
				logger.Error(fmt.Sprintf("Backup failed: %v", err))
			}
		}
	}
}

func (s *Scheduler) doBackup() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	backupPath := s.backupPath()

	if err := VacuumInto(s.db, backupPath); err != nil {
		s.lastError = err
		return err
	}

	s.lastBackup = time.Now()
	s.lastError = nil

	logger.Info(fmt.Sprintf("Backup created: %s", backupPath))

	// Prune old backups.
	if s.cfg.MaxBackups > 0 {
		if err := pruneBackups(s.cfg.BackupDir, s.cfg.MaxBackups); err != nil {
			logger.Warn(fmt.Sprintf("Backup prune failed: %v", err))
		}
	}

	return nil
}

func (s *Scheduler) backupPath() string {
	ts := time.Now().UTC().Format("20060102-150405")
	base := filepath.Base(s.cfg.DBPath)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	return filepath.Join(s.cfg.BackupDir, fmt.Sprintf("%s-%s.db", name, ts))
}

// VacuumInto creates an atomic backup of the database at destPath using
// SQLite's VACUUM INTO statement. The destination must not exist.
func VacuumInto(db *sql.DB, destPath string) error {
	// VACUUM INTO creates a full, consistent copy in a single statement.
	_, err := db.Exec("VACUUM INTO ?", destPath)
	if err != nil {
		// Clean up partial file if it was created.
		os.Remove(destPath)
		return fmt.Errorf("vacuum into %q: %w", destPath, err)
	}

	// Verify the backup file exists and has content.
	info, err := os.Stat(destPath)
	if err != nil {
		return fmt.Errorf("verify backup %q: %w", destPath, err)
	}
	if info.Size() == 0 {
		os.Remove(destPath)
		return fmt.Errorf("backup %q is empty", destPath)
	}

	return nil
}

// pruneBackups removes the oldest .db files in dir until at most maxKeep remain.
func pruneBackups(dir string, maxKeep int) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read backup dir %q: %w", dir, err)
	}

	// Collect only .db backup files.
	var backups []os.DirEntry
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".db") {
			backups = append(backups, e)
		}
	}

	if len(backups) <= maxKeep {
		return nil
	}

	// Sort by name ascending (timestamp-based names sort chronologically).
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Name() < backups[j].Name()
	})

	// Remove oldest files.
	toRemove := len(backups) - maxKeep
	for i := 0; i < toRemove; i++ {
		path := filepath.Join(dir, backups[i].Name())
		if err := os.Remove(path); err != nil {
			logger.Warn(fmt.Sprintf("Failed to remove old backup %q: %v", path, err))
		} else {
			logger.Info(fmt.Sprintf("Pruned old backup: %s", backups[i].Name()))
		}
	}

	return nil
}

// ListBackups returns the backup files in the given directory, sorted oldest first.
func ListBackups(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read backup dir %q: %w", dir, err)
	}

	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".db") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}
