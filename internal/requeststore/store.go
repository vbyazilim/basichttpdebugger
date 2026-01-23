package requeststore

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

const defaultMaxSize = 50

// Request represents a captured HTTP request.
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

// Store holds captured requests in memory with pub/sub support for SSE.
type Store struct {
	mu        sync.RWMutex
	requests  []Request
	maxSize   int
	listeners []chan Request
}

// New creates a new request store with the given max size.
func New(maxSize int) *Store {
	if maxSize <= 0 {
		maxSize = defaultMaxSize
	}

	return &Store{
		requests:  make([]Request, 0, maxSize),
		maxSize:   maxSize,
		listeners: make([]chan Request, 0),
	}
}

// Add adds a new request to the store and broadcasts to listeners.
func (s *Store) Add(req Request) {
	if req.ID == "" {
		req.ID = uuid.New().String()
	}

	s.mu.Lock()

	if len(s.requests) >= s.maxSize {
		s.requests = s.requests[1:]
	}

	s.requests = append(s.requests, req)

	listeners := make([]chan Request, len(s.listeners))
	copy(listeners, s.listeners)

	s.mu.Unlock()

	for _, ch := range listeners {
		select {
		case ch <- req:
		default:
		}
	}
}

// GetAll returns all stored requests, newest first.
func (s *Store) GetAll() []Request {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Request, len(s.requests))
	for i, req := range s.requests {
		result[len(s.requests)-1-i] = req
	}

	return result
}

// Count returns the number of stored requests.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.requests)
}

// Subscribe creates a new channel for receiving new requests.
func (s *Store) Subscribe() chan Request {
	ch := make(chan Request, 10)

	s.mu.Lock()
	s.listeners = append(s.listeners, ch)
	s.mu.Unlock()

	return ch
}

// Unsubscribe removes a channel from the listeners list.
func (s *Store) Unsubscribe(ch chan Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, listener := range s.listeners {
		if listener != ch {
			continue
		}

		s.listeners = append(s.listeners[:i], s.listeners[i+1:]...)
		close(ch)

		break
	}
}

// ListenerCount returns the number of active listeners.
func (s *Store) ListenerCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.listeners)
}
