package client

import "errors"

const discoveryAppURL = "https://prod-s0-webapp-proxy.nubank.com.br/api/app/discovery"

var ErrStatusCodeUnexpected = errors.New("unexpected status code")
var ErrNotAuthenticated = errors.New("client not authenticated. Did you called Authenticate?")
