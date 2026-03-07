package integrations

import "crypto/rand"

// _cryptoRand wraps crypto/rand.Read.
func _cryptoRand(b []byte) (int, error) {
	return rand.Read(b)
}
