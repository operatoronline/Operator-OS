package backup

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

// newTestDB creates a temporary SQLite database with test data.
func newTestDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(wal)")
	require.NoError(t, err)

	_, err = db.Exec(`CREATE TABLE test (id INTEGER PRIMARY KEY, value TEXT)`)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO test (value) VALUES ('hello'), ('world')`)
	require.NoError(t, err)

	return db, dbPath
}

func TestVacuumInto(t *testing.T) {
	db, _ := newTestDB(t)
	defer db.Close()

	destPath := filepath.Join(t.TempDir(), "backup.db")
	err := VacuumInto(db, destPath)
	require.NoError(t, err)

	// Verify backup exists and has data.
	info, err := os.Stat(destPath)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))

	// Open backup and verify contents.
	backupDB, err := sql.Open("sqlite", destPath)
	require.NoError(t, err)
	defer backupDB.Close()

	var count int
	err = backupDB.QueryRow("SELECT COUNT(*) FROM test").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestVacuumInto_DestExists(t *testing.T) {
	db, _ := newTestDB(t)
	defer db.Close()

	destPath := filepath.Join(t.TempDir(), "backup.db")

	// First backup succeeds.
	err := VacuumInto(db, destPath)
	require.NoError(t, err)

	// Second backup to same path should fail (file exists).
	err = VacuumInto(db, destPath)
	assert.Error(t, err)
}

func TestNewScheduler_Validation(t *testing.T) {
	db, dbPath := newTestDB(t)
	defer db.Close()

	tests := []struct {
		name string
		cfg  Config
		err  string
	}{
		{
			name: "missing BackupDir",
			cfg:  Config{DBPath: dbPath, Interval: time.Hour},
			err:  "BackupDir is required",
		},
		{
			name: "missing DBPath",
			cfg:  Config{BackupDir: t.TempDir(), Interval: time.Hour},
			err:  "DBPath is required",
		},
		{
			name: "zero interval",
			cfg:  Config{DBPath: dbPath, BackupDir: t.TempDir(), Interval: 0},
			err:  "Interval must be positive",
		},
		{
			name: "negative interval",
			cfg:  Config{DBPath: dbPath, BackupDir: t.TempDir(), Interval: -time.Hour},
			err:  "Interval must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewScheduler(db, tt.cfg)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.err)
		})
	}
}

func TestNewScheduler_CreatesDir(t *testing.T) {
	db, dbPath := newTestDB(t)
	defer db.Close()

	backupDir := filepath.Join(t.TempDir(), "nested", "backup", "dir")
	_, err := NewScheduler(db, Config{
		DBPath:    dbPath,
		BackupDir: backupDir,
		Interval:  time.Hour,
	})
	require.NoError(t, err)

	info, err := os.Stat(backupDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestScheduler_RunOnce(t *testing.T) {
	db, dbPath := newTestDB(t)
	defer db.Close()

	backupDir := filepath.Join(t.TempDir(), "backups")
	sched, err := NewScheduler(db, Config{
		DBPath:     dbPath,
		BackupDir:  backupDir,
		Interval:   time.Hour,
		MaxBackups: 5,
	})
	require.NoError(t, err)

	err = sched.RunOnce()
	require.NoError(t, err)

	// Verify backup was created.
	assert.False(t, sched.LastBackup().IsZero())
	assert.NoError(t, sched.LastError())

	backups, err := ListBackups(backupDir)
	require.NoError(t, err)
	assert.Len(t, backups, 1)
	assert.Contains(t, backups[0], "test-")
	assert.Contains(t, backups[0], ".db")
}

func TestScheduler_StartStop(t *testing.T) {
	db, dbPath := newTestDB(t)
	defer db.Close()

	backupDir := filepath.Join(t.TempDir(), "backups")
	sched, err := NewScheduler(db, Config{
		DBPath:     dbPath,
		BackupDir:  backupDir,
		Interval:   time.Hour,
		MaxBackups: 5,
	})
	require.NoError(t, err)

	sched.Start()
	// Stop should not hang.
	sched.Stop()
}

func TestPruneBackups(t *testing.T) {
	dir := t.TempDir()

	// Create 5 fake backup files with sorted names.
	for i := 1; i <= 5; i++ {
		name := filepath.Join(dir, "test-20260301-00000"+string(rune('0'+i))+".db")
		require.NoError(t, os.WriteFile(name, []byte("data"), 0o644))
	}

	// Prune to keep 3.
	err := pruneBackups(dir, 3)
	require.NoError(t, err)

	remaining, err := ListBackups(dir)
	require.NoError(t, err)
	assert.Len(t, remaining, 3)

	// The 2 oldest should be gone (001, 002), keeping 003, 004, 005.
	assert.Contains(t, remaining[0], "000003")
	assert.Contains(t, remaining[1], "000004")
	assert.Contains(t, remaining[2], "000005")
}

func TestPruneBackups_UnderLimit(t *testing.T) {
	dir := t.TempDir()

	// Create 2 files, prune to 5 — nothing removed.
	for i := 1; i <= 2; i++ {
		name := filepath.Join(dir, "test-2026030"+string(rune('0'+i))+".db")
		require.NoError(t, os.WriteFile(name, []byte("data"), 0o644))
	}

	err := pruneBackups(dir, 5)
	require.NoError(t, err)

	remaining, err := ListBackups(dir)
	require.NoError(t, err)
	assert.Len(t, remaining, 2)
}

func TestPruneBackups_IgnoresNonDB(t *testing.T) {
	dir := t.TempDir()

	// Mix of .db and other files.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "backup-001.db"), []byte("d"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "backup-002.db"), []byte("d"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "backup-003.db"), []byte("d"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("d"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "readme.md"), []byte("d"), 0o644))

	err := pruneBackups(dir, 2)
	require.NoError(t, err)

	remaining, err := ListBackups(dir)
	require.NoError(t, err)
	assert.Len(t, remaining, 2)

	// Non-DB files should still be there.
	entries, _ := os.ReadDir(dir)
	assert.Len(t, entries, 4) // 2 .db + 2 other
}

func TestListBackups_Empty(t *testing.T) {
	dir := t.TempDir()
	backups, err := ListBackups(dir)
	require.NoError(t, err)
	assert.Empty(t, backups)
}

func TestListBackups_Sorted(t *testing.T) {
	dir := t.TempDir()

	// Create files out of order.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "db-20260303.db"), []byte("d"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "db-20260301.db"), []byte("d"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "db-20260302.db"), []byte("d"), 0o644))

	backups, err := ListBackups(dir)
	require.NoError(t, err)
	assert.Equal(t, []string{"db-20260301.db", "db-20260302.db", "db-20260303.db"}, backups)
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, 6*time.Hour, cfg.Interval)
	assert.Equal(t, 7, cfg.MaxBackups)
	assert.Empty(t, cfg.DBPath)
	assert.Empty(t, cfg.BackupDir)
}

func TestScheduler_MultipleRunOnce(t *testing.T) {
	db, dbPath := newTestDB(t)
	defer db.Close()

	backupDir := filepath.Join(t.TempDir(), "backups")
	sched, err := NewScheduler(db, Config{
		DBPath:     dbPath,
		BackupDir:  backupDir,
		Interval:   time.Hour,
		MaxBackups: 3,
	})
	require.NoError(t, err)

	// Run multiple backups with a small delay to get different timestamps.
	for i := 0; i < 5; i++ {
		time.Sleep(time.Second) // ensure unique timestamp
		err = sched.RunOnce()
		require.NoError(t, err)
	}

	// Should be pruned to 3.
	backups, err := ListBackups(backupDir)
	require.NoError(t, err)
	assert.Len(t, backups, 3)
}

func TestScheduler_BackupContents(t *testing.T) {
	db, dbPath := newTestDB(t)
	defer db.Close()

	backupDir := filepath.Join(t.TempDir(), "backups")
	sched, err := NewScheduler(db, Config{
		DBPath:     dbPath,
		BackupDir:  backupDir,
		Interval:   time.Hour,
		MaxBackups: 5,
	})
	require.NoError(t, err)

	// Add more data after creating the scheduler.
	_, err = db.Exec(`INSERT INTO test (value) VALUES ('extra')`)
	require.NoError(t, err)

	err = sched.RunOnce()
	require.NoError(t, err)

	// Open the backup and verify it has the new data.
	backups, err := ListBackups(backupDir)
	require.NoError(t, err)
	require.Len(t, backups, 1)

	backupPath := filepath.Join(backupDir, backups[0])
	backupDB, err := sql.Open("sqlite", backupPath)
	require.NoError(t, err)
	defer backupDB.Close()

	var count int
	err = backupDB.QueryRow("SELECT COUNT(*) FROM test").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 3, count) // original 2 + 1 new
}
