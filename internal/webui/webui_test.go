package webui

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vbyazilim/basichttpdebugger/internal/requeststore"
)

type mockResponseWriter struct {
	header http.Header
	writer io.Writer
}

func (m *mockResponseWriter) Header() http.Header {
	return m.header
}

func (m *mockResponseWriter) Write(b []byte) (int, error) {
	return m.writer.Write(b)
}

func (m *mockResponseWriter) WriteHeader(_ int) {}

func (m *mockResponseWriter) Flush() {}

func TestNew(t *testing.T) {
	store := requeststore.New(50)
	webui := New(store, ":9003", ":9002")

	assert.NotNil(t, webui)
	assert.Equal(t, ":9003", webui.ListenAddr())
}

func TestBuildDebugURL(t *testing.T) {
	tests := []struct {
		name      string
		debugAddr string
		path      string
		expected  string
	}{
		{
			name:      "port only format",
			debugAddr: ":9002",
			path:      "/webhook",
			expected:  "http://localhost:9002/webhook",
		},
		{
			name:      "host and port format",
			debugAddr: "127.0.0.1:9002",
			path:      "/webhook",
			expected:  "http://127.0.0.1:9002/webhook",
		},
		{
			name:      "ipv6 localhost",
			debugAddr: "[::1]:9002",
			path:      "/api/test",
			expected:  "http://[::1]:9002/api/test",
		},
		{
			name:      "custom host",
			debugAddr: "0.0.0.0:8080",
			path:      "/",
			expected:  "http://0.0.0.0:8080/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildDebugURL(tt.debugAddr, tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWebUI_dashboardHandler(t *testing.T) {
	store := requeststore.New(50)
	webui := New(store, ":9003", ":9002")

	t.Run("serves index.html at root", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		webui.dashboardHandler(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "text/html")
		assert.Contains(t, rec.Body.String(), "<!DOCTYPE html>")
	})

	t.Run("returns 404 for other paths", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/other", nil)
		rec := httptest.NewRecorder()

		webui.dashboardHandler(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestWebUI_requestsHandler(t *testing.T) {
	t.Run("returns empty array when no requests", func(t *testing.T) {
		store := requeststore.New(50)
		webui := New(store, ":9003", ":9002")

		req := httptest.NewRequest(http.MethodGet, "/api/requests", nil)
		rec := httptest.NewRecorder()

		webui.requestsHandler(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")
		assert.Equal(t, "[]\n", rec.Body.String())
	})

	t.Run("returns requests as JSON", func(t *testing.T) {
		store := requeststore.New(50)
		store.Add(requeststore.Request{
			ID:     "test-1",
			Method: "POST",
			URL:    "/webhook",
		})
		webui := New(store, ":9003", ":9002")

		req := httptest.NewRequest(http.MethodGet, "/api/requests", nil)
		rec := httptest.NewRecorder()

		webui.requestsHandler(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var requests []requeststore.Request
		err := json.Unmarshal(rec.Body.Bytes(), &requests)

		require.NoError(t, err)
		assert.Len(t, requests, 1)
		assert.Equal(t, "test-1", requests[0].ID)
		assert.Equal(t, "POST", requests[0].Method)
	})

	t.Run("rejects non-GET methods", func(t *testing.T) {
		store := requeststore.New(50)
		webui := New(store, ":9003", ":9002")

		req := httptest.NewRequest(http.MethodPost, "/api/requests", nil)
		rec := httptest.NewRecorder()

		webui.requestsHandler(rec, req)

		assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
	})
}

func TestWebUI_eventsHandler(t *testing.T) {
	t.Run("sets SSE headers", func(t *testing.T) {
		store := requeststore.New(50)
		webui := New(store, ":9003", ":9002")

		// Use mockResponseWriter which is written once before goroutine reads
		header := http.Header{}
		pr, pw := io.Pipe()

		rec := &mockResponseWriter{
			header: header,
			writer: pw,
		}

		req := httptest.NewRequest(http.MethodGet, "/events", nil)

		handlerDone := make(chan struct{})

		go func() {
			webui.eventsHandler(rec, req)
			close(handlerDone)
		}()

		// Wait for handler to set headers
		time.Sleep(50 * time.Millisecond)

		// Stop the webui to terminate the handler
		_ = webui.Stop()

		// Wait for handler to finish before reading headers
		<-handlerDone

		// Now safe to read headers
		assert.Equal(t, "text/event-stream", header.Get("Content-Type"))
		assert.Equal(t, "no-cache", header.Get("Cache-Control"))
		assert.Equal(t, "keep-alive", header.Get("Connection"))

		pr.Close()
		pw.Close()
	})

	t.Run("broadcasts new requests", func(t *testing.T) {
		store := requeststore.New(50)
		webui := New(store, ":9003", ":9002")

		pr, pw := io.Pipe()

		rec := &mockResponseWriter{
			header: http.Header{},
			writer: pw,
		}

		ctx, cancel := context.WithCancel(context.Background())
		req := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(ctx)

		go func() {
			webui.eventsHandler(rec, req)
		}()

		time.Sleep(50 * time.Millisecond)

		store.Add(requeststore.Request{
			ID:     "sse-test",
			Method: "GET",
			URL:    "/test",
		})

		reader := bufio.NewReader(pr)
		line, err := reader.ReadString('\n')
		require.NoError(t, err)

		assert.True(t, strings.HasPrefix(line, "data: "))
		assert.Contains(t, line, "sse-test")

		cancel()
		pw.Close()
	})

	t.Run("stops gracefully when webui context is cancelled", func(t *testing.T) {
		store := requeststore.New(50)
		webui := New(store, ":9003", ":9002")

		pr, pw := io.Pipe()

		rec := &mockResponseWriter{
			header: http.Header{},
			writer: pw,
		}

		req := httptest.NewRequest(http.MethodGet, "/events", nil)

		handlerDone := make(chan struct{})

		go func() {
			webui.eventsHandler(rec, req)
			close(handlerDone)
		}()

		time.Sleep(50 * time.Millisecond)

		// Stop should cancel the context and cause handler to exit
		err := webui.Stop()
		require.NoError(t, err)

		// Handler should exit quickly after Stop
		select {
		case <-handlerDone:
			// Success - handler exited
		case <-time.After(1 * time.Second):
			t.Fatal("handler did not exit after Stop()")
		}

		pr.Close()
		pw.Close()
	})
}

func TestWebUI_replayHandler(t *testing.T) {
	t.Run("rejects non-POST methods", func(t *testing.T) {
		store := requeststore.New(50)
		webui := New(store, ":9003", ":9002")

		req := httptest.NewRequest(http.MethodGet, "/api/replay", nil)
		rec := httptest.NewRecorder()

		webui.replayHandler(rec, req)

		assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
	})

	t.Run("returns bad request for invalid JSON", func(t *testing.T) {
		store := requeststore.New(50)
		webui := New(store, ":9003", ":9002")

		req := httptest.NewRequest(http.MethodPost, "/api/replay", strings.NewReader("invalid json"))
		rec := httptest.NewRecorder()

		webui.replayHandler(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("returns not found for unknown request ID", func(t *testing.T) {
		store := requeststore.New(50)
		webui := New(store, ":9003", ":9002")

		body := `{"id": "unknown-id"}`
		req := httptest.NewRequest(http.MethodPost, "/api/replay", strings.NewReader(body))
		rec := httptest.NewRecorder()

		webui.replayHandler(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("replays request successfully", func(t *testing.T) {
		// Create a mock debug server
		debugServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "/webhook", r.URL.Path)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			assert.NotEmpty(t, r.Header.Get("X-Replayed-From"))

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"success": true}`))
		}))
		defer debugServer.Close()

		store := requeststore.New(50)

		// Extract host:port from the test server URL
		debugAddr := strings.TrimPrefix(debugServer.URL, "http://")
		webui := New(store, ":9003", debugAddr)

		// Add a request to replay
		store.Add(requeststore.Request{
			ID:      "replay-test-1",
			Method:  "POST",
			URL:     "/webhook",
			Headers: map[string]string{"Content-Type": "application/json"},
			Body:    `{"data": "test"}`,
		})

		body := `{"id": "replay-test-1"}`
		req := httptest.NewRequest(http.MethodPost, "/api/replay", strings.NewReader(body))
		rec := httptest.NewRecorder()

		webui.replayHandler(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

		var response map[string]any
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(200), response["status"])
	})

	t.Run("returns bad gateway when debug server is unavailable", func(t *testing.T) {
		store := requeststore.New(50)
		// Point to a port that's not listening
		webui := New(store, ":9003", "127.0.0.1:59999")

		store.Add(requeststore.Request{
			ID:     "replay-fail",
			Method: "GET",
			URL:    "/test",
		})

		body := `{"id": "replay-fail"}`
		req := httptest.NewRequest(http.MethodPost, "/api/replay", strings.NewReader(body))
		rec := httptest.NewRecorder()

		webui.replayHandler(rec, req)

		assert.Equal(t, http.StatusBadGateway, rec.Code)
	})
}
