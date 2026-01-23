package webui

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/vbyazilim/basichttpdebugger/internal/requeststore"
)

//go:embed static/index.html
var staticFiles embed.FS

const (
	defReadTimeout       = 5 * time.Second
	defReadHeaderTimeout = 5 * time.Second
	defWriteTimeout      = 30 * time.Second
	defIdleTimeout       = 60 * time.Second
)

// WebUI represents the web dashboard server.
type WebUI struct {
	store      *requeststore.Store
	listenAddr string
	server     *http.Server
}

// New creates a new WebUI instance.
func New(store *requeststore.Store, listenAddr string) *WebUI {
	w := &WebUI{
		store:      store,
		listenAddr: listenAddr,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", w.dashboardHandler)
	mux.HandleFunc("/events", w.eventsHandler)
	mux.HandleFunc("/api/requests", w.requestsHandler)

	w.server = &http.Server{
		Addr:              listenAddr,
		Handler:           mux,
		ReadTimeout:       defReadTimeout,
		ReadHeaderTimeout: defReadHeaderTimeout,
		WriteTimeout:      defWriteTimeout,
		IdleTimeout:       defIdleTimeout,
	}

	return w
}

// Start starts the web dashboard server.
func (w *WebUI) Start() error {
	log.Printf("web dashboard available at http://localhost%s\n", w.listenAddr)

	if err := w.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("webui start error: %w", err)
	}

	return nil
}

// Stop stops the web dashboard server.
func (w *WebUI) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := w.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("webui stop error: %w", err)
	}

	return nil
}

// ListenAddr returns the listen address.
func (w *WebUI) ListenAddr() string {
	return w.listenAddr
}

func (*WebUI) dashboardHandler(rw http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(rw, r)

		return
	}

	content, err := staticFiles.ReadFile("static/index.html")
	if err != nil {
		http.Error(rw, "internal server error", http.StatusInternalServerError)

		return
	}

	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = rw.Write(content)
}

func (w *WebUI) eventsHandler(rw http.ResponseWriter, r *http.Request) {
	flusher, ok := rw.(http.Flusher)
	if !ok {
		http.Error(rw, "streaming unsupported", http.StatusInternalServerError)

		return
	}

	rw.Header().Set("Content-Type", "text/event-stream")
	rw.Header().Set("Cache-Control", "no-cache")
	rw.Header().Set("Connection", "keep-alive")
	rw.Header().Set("Access-Control-Allow-Origin", "*")

	ch := w.store.Subscribe()
	defer w.store.Unsubscribe(ch)

	for {
		select {
		case req := <-ch:
			data, err := json.Marshal(req)
			if err != nil {
				continue
			}

			fmt.Fprintf(rw, "data: %s\n\n", data)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (w *WebUI) requestsHandler(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)

		return
	}

	requests := w.store.GetAll()

	rw.Header().Set("Content-Type", "application/json")
	rw.Header().Set("Access-Control-Allow-Origin", "*")

	if err := json.NewEncoder(rw).Encode(requests); err != nil {
		http.Error(rw, "internal server error", http.StatusInternalServerError)

		return
	}
}
