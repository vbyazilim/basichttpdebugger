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
	webui := New(store, ":9003")

	assert.NotNil(t, webui)
	assert.Equal(t, ":9003", webui.ListenAddr())
}

func TestWebUI_dashboardHandler(t *testing.T) {
	store := requeststore.New(50)
	webui := New(store, ":9003")

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
		webui := New(store, ":9003")

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
		webui := New(store, ":9003")

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
		webui := New(store, ":9003")

		req := httptest.NewRequest(http.MethodPost, "/api/requests", nil)
		rec := httptest.NewRecorder()

		webui.requestsHandler(rec, req)

		assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
	})
}

func TestWebUI_eventsHandler(t *testing.T) {
	t.Run("sets SSE headers", func(t *testing.T) {
		store := requeststore.New(50)
		webui := New(store, ":9003")

		req := httptest.NewRequest(http.MethodGet, "/events", nil)
		rec := httptest.NewRecorder()

		done := make(chan struct{})

		go func() {
			webui.eventsHandler(rec, req)
			close(done)
		}()

		time.Sleep(50 * time.Millisecond)

		assert.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))
		assert.Equal(t, "no-cache", rec.Header().Get("Cache-Control"))
		assert.Equal(t, "keep-alive", rec.Header().Get("Connection"))
	})

	t.Run("broadcasts new requests", func(t *testing.T) {
		store := requeststore.New(50)
		webui := New(store, ":9003")

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
}
