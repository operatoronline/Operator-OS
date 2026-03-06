package state

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tempDBPath(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, "state.db")
}

func TestSQLiteStateStore_BasicGetSet(t *testing.T) {
	store, err := NewSQLiteStateStore(tempDBPath(t))
	require.NoError(t, err)
	defer store.Close()

	// Get non-existent key returns empty string.
	val, err := store.Get("missing")
	require.NoError(t, err)
	assert.Equal(t, "", val)

	// Set and Get.
	require.NoError(t, store.Set("last_channel", "telegram"))
	val, err = store.Get("last_channel")
	require.NoError(t, err)
	assert.Equal(t, "telegram", val)

	// Overwrite.
	require.NoError(t, store.Set("last_channel", "discord"))
	val, err = store.Get("last_channel")
	require.NoError(t, err)
	assert.Equal(t, "discord", val)
}

func TestSQLiteStateStore_Timestamp(t *testing.T) {
	store, err := NewSQLiteStateStore(tempDBPath(t))
	require.NoError(t, err)
	defer store.Close()

	// Non-existent key returns zero time.
	ts, err := store.GetTimestamp("missing")
	require.NoError(t, err)
	assert.True(t, ts.IsZero())

	before := time.Now().Add(-time.Second)
	require.NoError(t, store.Set("key1", "value1"))
	after := time.Now().Add(time.Second)

	ts, err = store.GetTimestamp("key1")
	require.NoError(t, err)
	assert.True(t, ts.After(before), "timestamp should be after before")
	assert.True(t, ts.Before(after), "timestamp should be before after")

	// Update should change timestamp.
	time.Sleep(10 * time.Millisecond)
	require.NoError(t, store.Set("key1", "value2"))

	ts2, err := store.GetTimestamp("key1")
	require.NoError(t, err)
	assert.True(t, !ts2.Before(ts), "updated timestamp should not be before original")
}

func TestSQLiteStateStore_MultipleKeys(t *testing.T) {
	store, err := NewSQLiteStateStore(tempDBPath(t))
	require.NoError(t, err)
	defer store.Close()

	require.NoError(t, store.Set("a", "1"))
	require.NoError(t, store.Set("b", "2"))
	require.NoError(t, store.Set("c", "3"))

	v, _ := store.Get("a")
	assert.Equal(t, "1", v)
	v, _ = store.Get("b")
	assert.Equal(t, "2", v)
	v, _ = store.Get("c")
	assert.Equal(t, "3", v)
}

func TestSQLiteStateStore_Persistence(t *testing.T) {
	dbPath := tempDBPath(t)

	store1, err := NewSQLiteStateStore(dbPath)
	require.NoError(t, err)
	require.NoError(t, store1.Set("channel", "slack"))
	require.NoError(t, store1.Close())

	// Re-open and verify data persists.
	store2, err := NewSQLiteStateStore(dbPath)
	require.NoError(t, err)
	defer store2.Close()

	val, err := store2.Get("channel")
	require.NoError(t, err)
	assert.Equal(t, "slack", val)
}

func TestSQLiteStateStore_ConcurrentAccess(t *testing.T) {
	store, err := NewSQLiteStateStore(tempDBPath(t))
	require.NoError(t, err)
	defer store.Close()

	var wg sync.WaitGroup
	for i := range 20 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", idx%5)
			val := fmt.Sprintf("val-%d", idx)
			assert.NoError(t, store.Set(key, val))
			_, err := store.Get(key)
			assert.NoError(t, err)
		}(i)
	}
	wg.Wait()

	// Verify at least the keys exist.
	for i := range 5 {
		key := fmt.Sprintf("key-%d", i)
		val, err := store.Get(key)
		require.NoError(t, err)
		assert.NotEmpty(t, val)
	}
}

func TestSQLiteStateStore_EmptyValue(t *testing.T) {
	store, err := NewSQLiteStateStore(tempDBPath(t))
	require.NoError(t, err)
	defer store.Close()

	require.NoError(t, store.Set("empty", ""))
	val, err := store.Get("empty")
	require.NoError(t, err)
	assert.Equal(t, "", val)

	// Timestamp should still be set even for empty values.
	ts, err := store.GetTimestamp("empty")
	require.NoError(t, err)
	assert.False(t, ts.IsZero())
}

func TestNewManagerWithStore(t *testing.T) {
	store, err := NewSQLiteStateStore(tempDBPath(t))
	require.NoError(t, err)
	defer store.Close()

	tmpDir := t.TempDir()
	sm := NewManagerWithStore(tmpDir, store)

	// Test SetLastChannel via store-backed manager.
	require.NoError(t, sm.SetLastChannel("telegram"))
	assert.Equal(t, "telegram", sm.GetLastChannel())

	// Verify it's in the store too.
	val, err := store.Get("last_channel")
	require.NoError(t, err)
	assert.Equal(t, "telegram", val)

	// Test SetLastChatID.
	require.NoError(t, sm.SetLastChatID("12345"))
	assert.Equal(t, "12345", sm.GetLastChatID())

	val, err = store.Get("last_chat_id")
	require.NoError(t, err)
	assert.Equal(t, "12345", val)

	// Timestamp should be updated.
	assert.False(t, sm.GetTimestamp().IsZero())
}

func TestNewManagerWithStore_LoadsExisting(t *testing.T) {
	store, err := NewSQLiteStateStore(tempDBPath(t))
	require.NoError(t, err)
	defer store.Close()

	// Pre-populate the store.
	require.NoError(t, store.Set("last_channel", "discord"))
	require.NoError(t, store.Set("last_chat_id", "99999"))

	tmpDir := t.TempDir()
	sm := NewManagerWithStore(tmpDir, store)

	assert.Equal(t, "discord", sm.GetLastChannel())
	assert.Equal(t, "99999", sm.GetLastChatID())
}

func TestNewManagerWithStore_NoJSONFiles(t *testing.T) {
	store, err := NewSQLiteStateStore(tempDBPath(t))
	require.NoError(t, err)
	defer store.Close()

	tmpDir := t.TempDir()
	sm := NewManagerWithStore(tmpDir, store)
	require.NoError(t, sm.SetLastChannel("test"))

	// No JSON state files should be created.
	stateFile := filepath.Join(tmpDir, "state", "state.json")
	_, err = os.Stat(stateFile)
	assert.True(t, os.IsNotExist(err), "JSON state file should not exist when using store backend")
}
