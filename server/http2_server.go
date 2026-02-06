package server

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type HTTP2Middleware func(http.Handler) http.Handler

func applyHTTP2Middlewares(handler http.Handler, middlewares ...HTTP2Middleware) http.Handler {
	for _, mw := range middlewares {
		handler = mw(handler)
	}
	return handler
}

type HTTPServerConfig struct {
	ServerAddress string `env:"HTTP_SERVER_ADDRESS" envDefault:"0.0.0.0:9440"`

	EnableTLS   bool   `env:"ENABLE_TLS" envDefault:"false"`
	TLSCertFile string `env:"TLS_CERT_FILE" envDefault:"server.crt"`
	TLSKeyFile  string `env:"TLS_KEY_FILE" envDefault:"server.key"`

	ReadHeaderTimeoutSeconds int `env:"READ_HEADER_TIMEOUT_SECONDS" envDefault:""`
	ReadTimeoutSeconds       int `env:"READ_TIMEOUT_SECONDS" envDefault:""`
	WriteTimeoutSeconds      int `env:"WRITE_TIMEOUT_SECONDS" envDefault:""`
}

func NewHTTPServer(ctx context.Context, cfg HTTPServerConfig, mux *http.ServeMux, middlewares ...HTTP2Middleware) *http.Server {
	return &http.Server{
		BaseContext:       func(_ net.Listener) context.Context { return ctx },
		ReadHeaderTimeout: time.Duration(cfg.ReadHeaderTimeoutSeconds) * time.Second,
		ReadTimeout:       time.Duration(cfg.ReadTimeoutSeconds) * time.Second,
		WriteTimeout:      time.Duration(cfg.WriteTimeoutSeconds) * time.Second,
		Addr:              cfg.ServerAddress,
		Handler:           applyHTTP2Middlewares(mux, middlewares...),
	}
}

func NewHTTP2Server(ctx context.Context, cfg HTTPServerConfig, mux *http.ServeMux, middlewares ...HTTP2Middleware) (*http.Server, error) {
	if cfg.EnableTLS {
		return NewHTTP2ServerWithTLS(ctx, cfg, mux)
	}

	server := NewHTTPServer(ctx, cfg, mux, middlewares...)

	// Preserve middlewares when serving cleartext HTTP/2 (h2c)
	server.Handler = h2c.NewHandler(server.Handler, &http2.Server{})

	return server, nil
}

func NewHTTP2ServerWithTLS(ctx context.Context, cfg HTTPServerConfig, mux *http.ServeMux, middlewares ...HTTP2Middleware) (*http.Server, error) {
	if cfg.EnableTLS {
		return nil, ErrTLSDisabledForTLSOnlyServer
	}

	server := NewHTTPServer(ctx, cfg, mux, middlewares...)

	cert, err := tls.LoadX509KeyPair(cfg.TLSCertFile, cfg.TLSKeyFile)
	if err != nil {
		return nil, err
	}

	server.TLSConfig = &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	return server, nil
}

func IsHTTP2ServerWithTLS(server *http.Server) bool {
	return server.TLSConfig != nil
}
