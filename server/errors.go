package adapters

import "errors"

var (
	ErrTLSDisabledForTLSOnlyServer = errors.New("TLS is disabled for a TLS-only server")
)
