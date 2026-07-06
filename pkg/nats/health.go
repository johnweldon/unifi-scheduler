package nats

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	_ "net/http/pprof" // registers profiling handlers on http.DefaultServeMux
	"time"
)

// HealthzHandler returns an HTTP handler that reports 200 OK when live returns
// true and 503 Service Unavailable otherwise. It is intended to back Kubernetes
// liveness/readiness probes for the otherwise headless NATS agent.
func HealthzHandler(live func() bool) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if live != nil && live() {
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, "ok\n")
			return
		}

		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, "unavailable\n")
	}
}

// StartHealthServer starts an HTTP server on addr that serves GET /healthz backed
// by live, alongside the net/http/pprof handlers registered on the default mux. It
// shuts down when ctx is cancelled. If addr is empty the server is not started,
// preserving the agent's default headless behavior.
func StartHealthServer(ctx context.Context, addr string, live func() bool) {
	if addr == "" {
		return
	}

	http.HandleFunc("/healthz", HealthzHandler(live))

	srv := &http.Server{
		Addr:              addr,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("health server shutdown: %v", err)
		}
	}()

	go func() {
		log.Printf("health server listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("health server error: %v", err)
		}
	}()
}
