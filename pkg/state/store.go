package state

import "time"

// StateStore defines the persistence interface for application state.
// Implementations must be safe for concurrent use.
type StateStore interface {
	// Get retrieves the value for the given key. Returns "" if not found.
	Get(key string) (string, error)

	// Set stores a key-value pair with the current timestamp.
	Set(key string, value string) error

	// GetTimestamp returns the last update time for the given key.
	// Returns zero time if the key has never been set.
	GetTimestamp(key string) (time.Time, error)

	// Close releases any underlying resources.
	Close() error
}
