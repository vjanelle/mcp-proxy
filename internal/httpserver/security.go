package httpserver

import (
	"crypto/subtle"
	"net/http"
	"time"
)

const (
	defaultReadHeaderTimeout = 3 * time.Second
	defaultReadTimeout       = 15 * time.Second
	defaultWriteTimeout      = 30 * time.Second
	defaultIdleTimeout       = 60 * time.Second
	defaultMaxHeaderBytes    = 1 << 20 // 1 MiB
	defaultMaxRPCBodyBytes   = 1 << 20 // 1 MiB

	apiKeyHeader = "X-API-Key"
)

// SecurityOptions controls HTTP hardening and auth behavior.
type SecurityOptions struct {
	APIKey          string
	MaxRPCBodyBytes int64
}

func (o SecurityOptions) normalized() SecurityOptions {
	if o.MaxRPCBodyBytes <= 0 {
		o.MaxRPCBodyBytes = defaultMaxRPCBodyBytes
	}
	return o
}

func requireAPIKey(apiKey string, next http.HandlerFunc) http.HandlerFunc {
	if apiKey == "" {
		return next
	}

	return func(writer http.ResponseWriter, request *http.Request) {
		got := request.Header.Get(apiKeyHeader)
		if subtle.ConstantTimeCompare([]byte(got), []byte(apiKey)) != 1 {
			http.Error(writer, "unauthorized", http.StatusUnauthorized)
			return
		}

		next(writer, request)
	}
}
