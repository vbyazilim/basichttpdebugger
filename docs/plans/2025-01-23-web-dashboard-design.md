# Web Dashboard Feature Design

## Overview

Add a web-based dashboard to Basic HTTP Debugger that allows users to monitor incoming HTTP requests in real-time through a browser, similar to ngrok's web interface.

## Requirements

- Real-time updates via SSE (Server-Sent Events)
- Embedded HTML/CSS/JS (single binary deployment)
- Store last 50 requests in memory
- Web port = debug port + 1 (e.g., debug :9002 → web :9003)
- Display: time, method, URL, headers, body
- No changes to existing terminal functionality

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    DebugServer                          │
├─────────────────────────────────────────────────────────┤
│  :9002 (debug port)        :9003 (web port)             │
│  ┌─────────────────┐       ┌─────────────────┐          │
│  │ Debug Handler   │       │ Web Dashboard   │          │
│  │ (existing)      │──────▶│ - GET /         │          │
│  └─────────────────┘       │ - GET /events   │ (SSE)    │
│          │                 │ - GET /api/...  │          │
│          │                 └─────────────────┘          │
│          ▼                         ▲                    │
│  ┌─────────────────────────────────┴───┐                │
│  │         RequestStore (in-memory)    │                │
│  │         - Last 50 requests          │                │
│  │         - Thread-safe (sync.Mutex)  │                │
│  └─────────────────────────────────────┘                │
└─────────────────────────────────────────────────────────┘
```

## New Packages

### internal/requeststore

```go
type Request struct {
    ID      string            `json:"id"`
    Time    time.Time         `json:"time"`
    Method  string            `json:"method"`
    URL     string            `json:"url"`
    Headers map[string]string `json:"headers"`
    Body    string            `json:"body"`
    Host    string            `json:"host"`
    Proto   string            `json:"proto"`
}

type Store struct {
    mu        sync.RWMutex
    requests  []Request
    maxSize   int
    listeners []chan Request
}

func New(maxSize int) *Store
func (s *Store) Add(req Request)
func (s *Store) GetAll() []Request
func (s *Store) Subscribe() chan Request
func (s *Store) Unsubscribe(ch chan Request)
```

### internal/webui

```go
type WebUI struct {
    store      *requeststore.Store
    listenAddr string
    server     *http.Server
}

// Endpoints:
// GET /            → Dashboard HTML (embedded)
// GET /events      → SSE stream
// GET /api/requests → Current requests JSON
```

## File Structure

```
internal/
├── httpserver/
│   ├── httpserver.go      ← MODIFY (store integration)
│   └── run.go             ← MODIFY (webui startup)
├── requeststore/          ← NEW
│   ├── store.go
│   └── store_test.go
└── webui/                 ← NEW
    ├── webui.go
    ├── webui_test.go
    └── static/
        └── index.html
```

## Changes to Existing Code

### httpserver.go
- Add `Store *requeststore.Store` field to DebugServer
- Add `WithStore(s *Store) Option` function
- Call `store.Add(...)` at end of debugHandlerFunc

### run.go
- Create RequestStore
- Create and start WebUI (separate goroutine)
- Log: `web dashboard available at http://localhost:PORT`
- Add WebUI to graceful shutdown

## Web UI Layout

```
┌─────────────────────────────────────────────────────────┐
│  Basic HTTP Debugger - Web Dashboard        v0.4.1     │
├─────────────────────────────────────────────────────────┤
│  ● Connected                                            │
├──────────┬──────────────────────────────────────────────┤
│ REQUESTS │           REQUEST DETAIL                     │
│──────────│──────────────────────────────────────────────│
│ POST /x  │  Time: 2024-01-23 14:32:01 UTC              │
│ 14:32:01 │  Method: POST                               │
│          │  URL: /webhook                              │
│ GET /y   │  Headers: ...                               │
│ 14:31:45 │  Body: { ... }                              │
└──────────┴──────────────────────────────────────────────┘
```

## Dependencies

No new external dependencies. Uses only stdlib:
- `embed` for static files
- `encoding/json` for API
- `sync` for thread safety
