package requeststore

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("creates store with given max size", func(t *testing.T) {
		store := New(10)

		assert.NotNil(t, store)
		assert.Equal(t, 10, store.maxSize)
		assert.Empty(t, store.requests)
	})

	t.Run("uses default max size when zero", func(t *testing.T) {
		store := New(0)

		assert.Equal(t, defaultMaxSize, store.maxSize)
	})

	t.Run("uses default max size when negative", func(t *testing.T) {
		store := New(-1)

		assert.Equal(t, defaultMaxSize, store.maxSize)
	})
}

func TestStore_Add(t *testing.T) {
	t.Run("adds request to store", func(t *testing.T) {
		store := New(10)
		req := Request{
			Method: "GET",
			URL:    "/test",
			Time:   time.Now(),
		}

		store.Add(req)

		assert.Equal(t, 1, store.Count())
	})

	t.Run("generates ID if empty", func(t *testing.T) {
		store := New(10)
		req := Request{Method: "GET", URL: "/test"}

		store.Add(req)

		requests := store.GetAll()
		assert.NotEmpty(t, requests[0].ID)
	})

	t.Run("preserves ID if provided", func(t *testing.T) {
		store := New(10)
		req := Request{ID: "custom-id", Method: "GET", URL: "/test"}

		store.Add(req)

		requests := store.GetAll()
		assert.Equal(t, "custom-id", requests[0].ID)
	})

	t.Run("removes oldest when max size exceeded", func(t *testing.T) {
		store := New(3)
		store.Add(Request{ID: "1", Method: "GET", URL: "/first"})
		store.Add(Request{ID: "2", Method: "GET", URL: "/second"})
		store.Add(Request{ID: "3", Method: "GET", URL: "/third"})
		store.Add(Request{ID: "4", Method: "GET", URL: "/fourth"})

		assert.Equal(t, 3, store.Count())

		requests := store.GetAll()
		assert.Equal(t, "4", requests[0].ID)
		assert.Equal(t, "3", requests[1].ID)
		assert.Equal(t, "2", requests[2].ID)
	})
}

func TestStore_GetAll(t *testing.T) {
	t.Run("returns empty slice when no requests", func(t *testing.T) {
		store := New(10)

		requests := store.GetAll()

		assert.Empty(t, requests)
	})

	t.Run("returns requests newest first", func(t *testing.T) {
		store := New(10)
		store.Add(Request{ID: "1", Method: "GET", URL: "/first"})
		store.Add(Request{ID: "2", Method: "GET", URL: "/second"})
		store.Add(Request{ID: "3", Method: "GET", URL: "/third"})

		requests := store.GetAll()

		assert.Len(t, requests, 3)
		assert.Equal(t, "3", requests[0].ID)
		assert.Equal(t, "2", requests[1].ID)
		assert.Equal(t, "1", requests[2].ID)
	})
}

func TestStore_Subscribe(t *testing.T) {
	t.Run("creates channel and adds to listeners", func(t *testing.T) {
		store := New(10)

		ch := store.Subscribe()

		assert.NotNil(t, ch)
		assert.Equal(t, 1, store.ListenerCount())
	})

	t.Run("receives new requests on channel", func(t *testing.T) {
		store := New(10)
		ch := store.Subscribe()

		go func() {
			store.Add(Request{ID: "1", Method: "POST", URL: "/webhook"})
		}()

		select {
		case req := <-ch:
			assert.Equal(t, "1", req.ID)
			assert.Equal(t, "POST", req.Method)
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for request")
		}
	})
}

func TestStore_Unsubscribe(t *testing.T) {
	t.Run("removes channel from listeners", func(t *testing.T) {
		store := New(10)
		ch := store.Subscribe()

		store.Unsubscribe(ch)

		assert.Equal(t, 0, store.ListenerCount())
	})

	t.Run("no longer receives requests after unsubscribe", func(t *testing.T) {
		store := New(10)
		ch := store.Subscribe()

		store.Unsubscribe(ch)

		// Add a request - it should not be received on the unsubscribed channel
		store.Add(Request{ID: "after-unsub", Method: "GET", URL: "/test"})

		select {
		case <-ch:
			t.Fatal("should not receive on unsubscribed channel")
		case <-time.After(50 * time.Millisecond):
			// Expected - no message received
		}
	})
}

func TestStore_Concurrency(t *testing.T) {
	store := New(50)
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)

		go func(n int) {
			defer wg.Done()

			for j := 0; j < 10; j++ {
				store.Add(Request{Method: "GET", URL: "/test"})
			}
		}(i)
	}

	wg.Wait()

	assert.Equal(t, 50, store.Count())
}

func TestStore_BroadcastToMultipleListeners(t *testing.T) {
	store := New(10)
	ch1 := store.Subscribe()
	ch2 := store.Subscribe()

	var wg sync.WaitGroup

	wg.Add(2)

	received1 := make(chan Request, 1)
	received2 := make(chan Request, 1)

	go func() {
		defer wg.Done()

		select {
		case req := <-ch1:
			received1 <- req
		case <-time.After(time.Second):
		}
	}()

	go func() {
		defer wg.Done()

		select {
		case req := <-ch2:
			received2 <- req
		case <-time.After(time.Second):
		}
	}()

	store.Add(Request{ID: "broadcast-test", Method: "GET", URL: "/test"})

	wg.Wait()

	require.Len(t, received1, 1)
	require.Len(t, received2, 1)

	req1 := <-received1
	req2 := <-received2

	assert.Equal(t, "broadcast-test", req1.ID)
	assert.Equal(t, "broadcast-test", req2.ID)
}
