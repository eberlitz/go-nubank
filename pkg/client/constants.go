package client

import "errors"

const discoveryAppURL = "https://prod-s0-webapp-proxy.nubank.com.br/api/app/discovery"

var (
	ErrStatusCodeUnexpected = errors.New("unexpected status code")
	ErrNotAuthenticated     = errors.New("client not authenticated. Did you called Authenticate?")
	ErrNoEncryptedCode      = errors.New("no encrypted code found. Did you call `RequestCode` before exchanging certs?")
)
