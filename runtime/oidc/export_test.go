package oidc

import "time"

// SetKeysRetryForTest changes the retry gap of the JWKS and returns the undo.
func SetKeysRetryForTest(d time.Duration) func() {
	old := keysMinRetry
	keysMinRetry = d
	return func() { keysMinRetry = old }
}
