package auth

// CredentialStore abstracts credential persistence.
// Implementations must be safe for concurrent use.
type CredentialStore interface {
	// Get returns the credential for the given provider, or nil if not found.
	Get(provider string) (*AuthCredential, error)
	// Set stores (or updates) the credential for the given provider.
	Set(provider string, cred *AuthCredential) error
	// Delete removes the credential for the given provider.
	Delete(provider string) error
	// DeleteAll removes all stored credentials.
	DeleteAll() error
	// List returns all stored credentials keyed by provider name.
	List() (map[string]*AuthCredential, error)
	// Close releases any resources held by the store.
	Close() error
}
