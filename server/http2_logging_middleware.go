package server

import (
	"net/http"

	"github.com/northconn/go-lib/telemetry/log"
)

func HTTP2ServerWithLogging(wideEventName string) HTTP2Middleware {

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			correlationID := GetHTTP2CorrelationID(ctx)

			if correlationID == "" {
				correlationID = "unknown"
			}

			log.FromContext(ctx).With("correlation_id", correlationID, "method", r.Method, "path", r.URL.Path)

			ctx, wideEvent := log.EnsureWideEventFromContext(ctx, wideEventName)

			next.ServeHTTP(w, r.WithContext(ctx))

			wideEvent.Commit(ctx)
		})
	}

}
