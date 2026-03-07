package dbmigrate

import (
	"database/sql"
	"io/fs"
	"testing"
	"testing/fstest"

	_ "modernc.org/sqlite"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:?_pragma=journal_mode(wal)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(on)")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func testFS() fs.FS {
	return fstest.MapFS{
		"m/001_create_users.sql": &fstest.MapFile{
			Data: []byte(`CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL);`),
		},
		"m/002_create_posts.sql": &fstest.MapFile{
			Data: []byte(`CREATE TABLE posts (id INTEGER PRIMARY KEY, user_id INTEGER REFERENCES users(id), body TEXT);`),
		},
	}
}

func TestNew_NilDB(t *testing.T) {
	_, err := New(nil, testFS(), "m")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db is nil")
}

func TestNew_BadDir(t *testing.T) {
	db := openTestDB(t)
	_, err := New(db, testFS(), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "read dir")
}

func TestNew_DuplicateVersions(t *testing.T) {
	dupeFS := fstest.MapFS{
		"m/001_a.sql": &fstest.MapFile{Data: []byte("SELECT 1;")},
		"m/001_b.sql": &fstest.MapFile{Data: []byte("SELECT 2;")},
	}
	db := openTestDB(t)
	_, err := New(db, dupeFS, "m")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate")
}

func TestUp_AppliesAll(t *testing.T) {
	db := openTestDB(t)
	m, err := New(db, testFS(), "m")
	require.NoError(t, err)

	applied, err := m.Up()
	require.NoError(t, err)
	assert.Equal(t, 2, applied)

	// Tables should exist.
	_, err = db.Exec(`INSERT INTO users (name) VALUES ('alice')`)
	assert.NoError(t, err)
	_, err = db.Exec(`INSERT INTO posts (user_id, body) VALUES (1, 'hello')`)
	assert.NoError(t, err)
}

func TestUp_Idempotent(t *testing.T) {
	db := openTestDB(t)
	m, err := New(db, testFS(), "m")
	require.NoError(t, err)

	n1, err := m.Up()
	require.NoError(t, err)
	assert.Equal(t, 2, n1)

	// Run again — should apply 0.
	n2, err := m.Up()
	require.NoError(t, err)
	assert.Equal(t, 0, n2)
}

func TestUp_Incremental(t *testing.T) {
	// Apply v1, then add v2 and apply again.
	fsV1 := fstest.MapFS{
		"m/001_create_users.sql": &fstest.MapFile{
			Data: []byte(`CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL);`),
		},
	}

	db := openTestDB(t)
	m1, err := New(db, fsV1, "m")
	require.NoError(t, err)
	n, err := m1.Up()
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	// Now with both.
	m2, err := New(db, testFS(), "m")
	require.NoError(t, err)
	n, err = m2.Up()
	require.NoError(t, err)
	assert.Equal(t, 1, n) // only v2 should apply
}

func TestApplied(t *testing.T) {
	db := openTestDB(t)
	m, err := New(db, testFS(), "m")
	require.NoError(t, err)

	// Before any migration.
	applied, err := m.Applied()
	require.NoError(t, err)
	assert.Empty(t, applied)

	_, err = m.Up()
	require.NoError(t, err)

	applied, err = m.Applied()
	require.NoError(t, err)
	assert.Len(t, applied, 2)
	assert.Equal(t, 1, applied[0].Version)
	assert.Equal(t, "001_create_users.sql", applied[0].Name)
	assert.Equal(t, 2, applied[1].Version)
	assert.False(t, applied[0].AppliedAt.IsZero())
}

func TestPending(t *testing.T) {
	db := openTestDB(t)
	m, err := New(db, testFS(), "m")
	require.NoError(t, err)

	pending, err := m.Pending()
	require.NoError(t, err)
	assert.Len(t, pending, 2)

	_, err = m.Up()
	require.NoError(t, err)

	pending, err = m.Pending()
	require.NoError(t, err)
	assert.Empty(t, pending)
}

func TestVersion(t *testing.T) {
	db := openTestDB(t)
	m, err := New(db, testFS(), "m")
	require.NoError(t, err)

	v, err := m.Version()
	require.NoError(t, err)
	assert.Equal(t, 0, v)

	_, err = m.Up()
	require.NoError(t, err)

	v, err = m.Version()
	require.NoError(t, err)
	assert.Equal(t, 2, v)
}

func TestUp_FailedMigration_Rolls_Back(t *testing.T) {
	badFS := fstest.MapFS{
		"m/001_good.sql": &fstest.MapFile{
			Data: []byte(`CREATE TABLE t1 (id INTEGER PRIMARY KEY);`),
		},
		"m/002_bad.sql": &fstest.MapFile{
			Data: []byte(`THIS IS NOT VALID SQL;`),
		},
	}

	db := openTestDB(t)
	m, err := New(db, badFS, "m")
	require.NoError(t, err)

	applied, err := m.Up()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "migration 2")
	assert.Equal(t, 1, applied) // v1 succeeded

	// v1 should be recorded, v2 should not.
	v, err := m.Version()
	require.NoError(t, err)
	assert.Equal(t, 1, v)

	// t1 should exist (v1 applied), but nothing from v2.
	_, err = db.Exec(`INSERT INTO t1 (id) VALUES (1)`)
	assert.NoError(t, err)
}

func TestNewFromList(t *testing.T) {
	db := openTestDB(t)
	m, err := NewFromList(db, []Migration{
		{Version: 2, Name: "two", SQL: `CREATE TABLE t2 (id INTEGER PRIMARY KEY);`},
		{Version: 1, Name: "one", SQL: `CREATE TABLE t1 (id INTEGER PRIMARY KEY);`},
	})
	require.NoError(t, err)

	n, err := m.Up()
	require.NoError(t, err)
	assert.Equal(t, 2, n)

	// Check ordering was correct by verifying tables exist.
	_, err = db.Exec(`INSERT INTO t1 (id) VALUES (1)`)
	assert.NoError(t, err)
	_, err = db.Exec(`INSERT INTO t2 (id) VALUES (1)`)
	assert.NoError(t, err)
}

func TestNewFromList_DuplicateVersion(t *testing.T) {
	db := openTestDB(t)
	_, err := NewFromList(db, []Migration{
		{Version: 1, Name: "a", SQL: "SELECT 1;"},
		{Version: 1, Name: "b", SQL: "SELECT 2;"},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate")
}

func TestNewFromList_NilDB(t *testing.T) {
	_, err := NewFromList(nil, nil)
	assert.Error(t, err)
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name    string
		version int
		err     bool
	}{
		{"001_create.sql", 1, false},
		{"42_add_index.sql", 42, false},
		{"100.sql", 100, false},
		{"abc_nope.sql", 0, true},
		{"_leading_underscore.sql", 0, true},
	}
	for _, tt := range tests {
		v, err := parseVersion(tt.name)
		if tt.err {
			assert.Error(t, err, "name=%s", tt.name)
		} else {
			require.NoError(t, err, "name=%s", tt.name)
			assert.Equal(t, tt.version, v, "name=%s", tt.name)
		}
	}
}

func TestNonSQLFilesIgnored(t *testing.T) {
	mixed := fstest.MapFS{
		"m/001_ok.sql":  &fstest.MapFile{Data: []byte(`CREATE TABLE t1 (id INTEGER PRIMARY KEY);`)},
		"m/README.md":   &fstest.MapFile{Data: []byte("# Migrations\n")},
		"m/notes.txt":   &fstest.MapFile{Data: []byte("some notes")},
		"m/.gitkeep":    &fstest.MapFile{Data: []byte("")},
	}

	db := openTestDB(t)
	m, err := New(db, mixed, "m")
	require.NoError(t, err)

	n, err := m.Up()
	require.NoError(t, err)
	assert.Equal(t, 1, n)
}

func TestAutoMigrate_WithEmbeddedMigrations(t *testing.T) {
	db := openTestDB(t)

	n, err := AutoMigrate(db)
	require.NoError(t, err)
	assert.Equal(t, 15, n) // 001–015

	// Verify all tables created.
	_, err = db.Exec(`INSERT INTO sessions (key) VALUES ('test')`)
	assert.NoError(t, err)
	_, err = db.Exec(`INSERT INTO state (key, value) VALUES ('k', 'v')`)
	assert.NoError(t, err)
	_, err = db.Exec(`INSERT INTO credentials (provider, encrypted_data) VALUES ('test', x'00')`)
	assert.NoError(t, err)

	// Verify tenant_id column exists (migration 005).
	_, err = db.Exec(`UPDATE sessions SET tenant_id = 'test-tenant' WHERE key = 'test'`)
	assert.NoError(t, err)

	// Idempotent.
	n, err = AutoMigrate(db)
	require.NoError(t, err)
	assert.Equal(t, 0, n)
}

func TestMultipleMigrations_OrderMatters(t *testing.T) {
	// Migrations where v2 depends on v1 (FK reference).
	ordered := fstest.MapFS{
		"m/001_parent.sql": &fstest.MapFile{
			Data: []byte(`CREATE TABLE parent (id INTEGER PRIMARY KEY);`),
		},
		"m/002_child.sql": &fstest.MapFile{
			Data: []byte(`CREATE TABLE child (id INTEGER PRIMARY KEY, parent_id INTEGER REFERENCES parent(id));`),
		},
	}

	db := openTestDB(t)
	m, err := New(db, ordered, "m")
	require.NoError(t, err)

	n, err := m.Up()
	require.NoError(t, err)
	assert.Equal(t, 2, n)
}
