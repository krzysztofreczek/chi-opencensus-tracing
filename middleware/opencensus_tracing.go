package middleware

import (
	"fmt"
	"net/http"

	"go.opencensus.io/trace"
)

// OpencensusTracing implements a simple middleware handler
// for adding an opencensus tracing span to the request context
func OpencensusTracing() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx, span := trace.StartSpan(r.Context(), spanName(r))
			defer span.End()

			next.ServeHTTP(w, r.WithContext(ctx))
		}

		return http.HandlerFunc(fn)
	}
}

func spanName(r *http.Request) string {
	return fmt.Sprintf("[%s] %s", r.Method, r.URL.String())
}
