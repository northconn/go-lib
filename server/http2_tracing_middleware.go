package server

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type http2CorrelationIdCtx struct{}

var http2CorrelationIdKey http2CorrelationIdCtx

func newHTTP2ContextWithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, http2CorrelationIdKey, correlationID)
}

func GetHTTP2CorrelationID(ctx context.Context) string {
	if correlationID, ok := ctx.Value(http2CorrelationIdKey).(string); ok {
		return correlationID
	}
	return ""
}

func HTTP2ServerWithTracing(requestIdHeaderKey string) HTTP2Middleware {

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var correlationID string

			if cID := r.Header.Get(requestIdHeaderKey); cID != "" {
				correlationID = cID
			} else {
				correlationID = uuid.New().String()
			}

			w.Header().Set(requestIdHeaderKey, correlationID)

			ctx := newHTTP2ContextWithCorrelationID(r.Context(), correlationID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

}
