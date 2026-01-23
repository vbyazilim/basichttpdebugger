package httpserver_test

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vbyazilim/basichttpdebugger/internal/httpserver"
)

func TestNew(t *testing.T) {
	t.Run("Default configuration", func(t *testing.T) {
		server, err := httpserver.New()
		require.NoError(t, err)
		assert.NotNil(t, server)
		assert.Equal(t, ":9002", server.ListenAddr)
		assert.NotNil(t, server.HTTPServer)
	})

	t.Run("With custom listen address", func(t *testing.T) {
		server, err := httpserver.New(
			httpserver.WithListenAddr(":8080"),
		)
		require.NoError(t, err)
		assert.Equal(t, ":8080", server.ListenAddr)
	})

	t.Run("With invalid listen address", func(t *testing.T) {
		server, err := httpserver.New(
			httpserver.WithListenAddr("invalid"),
		)
		assert.Error(t, err)
		assert.Nil(t, server)
		assert.Contains(t, err.Error(), "error listen addr")
	})

	t.Run("With invalid output writer", func(t *testing.T) {
		server, err := httpserver.New(
			httpserver.WithOutputWriter("/nonexistent/path/to/file.log"),
		)
		assert.Error(t, err)
		assert.Nil(t, server)
		assert.Contains(t, err.Error(), "invalid output")
	})

	t.Run("With HMAC configuration", func(t *testing.T) {
		server, err := httpserver.New(
			httpserver.WithHMACSecret("my-secret"),
			httpserver.WithHMACHeaderName("X-Hub-Signature-256"),
		)
		require.NoError(t, err)
		assert.Equal(t, "my-secret", server.HMACSecret)
		assert.Equal(t, "X-Hub-Signature-256", server.HMACHeaderName)
	})

	t.Run("With secret token configuration", func(t *testing.T) {
		server, err := httpserver.New(
			httpserver.WithSecretToken("my-token"),
			httpserver.WithSecretTokenHeaderName("X-Gitlab-Token"),
		)
		require.NoError(t, err)
		assert.Equal(t, "my-token", server.SecretToken)
		assert.Equal(t, "X-Gitlab-Token", server.SecretTokenHeaderName)
	})

	t.Run("With timeout configurations", func(t *testing.T) {
		server, err := httpserver.New(
			httpserver.WithReadTimeout(10*time.Second),
			httpserver.WithReadHeaderTimeout(10*time.Second),
			httpserver.WithWriteTimeout(20*time.Second),
			httpserver.WithIdleTimeout(30*time.Second),
		)
		require.NoError(t, err)
		assert.Equal(t, 10*time.Second, server.ReadTimeout)
		assert.Equal(t, 10*time.Second, server.ReadHeaderTimeout)
		assert.Equal(t, 20*time.Second, server.WriteTimeout)
		assert.Equal(t, 30*time.Second, server.IdleTimeout)
	})

	t.Run("With color enabled", func(t *testing.T) {
		server, err := httpserver.New(
			httpserver.WithColor(true),
		)
		require.NoError(t, err)
		assert.True(t, server.Color)
	})

	t.Run("With save raw HTTP request", func(t *testing.T) {
		server, err := httpserver.New(
			httpserver.WithSaveRawHTTPRequest(true),
			httpserver.WithRawHTTPRequestFileSaveFormat("%Y-%m-%d.raw"),
		)
		require.NoError(t, err)
		assert.True(t, server.SaveRawHTTPRequest)
		assert.Equal(t, "%Y-%m-%d.raw", server.RawHTTPRequestFileSaveFormat)
	})

	t.Run("With stdout output", func(t *testing.T) {
		server, err := httpserver.New(
			httpserver.WithOutputWriter("stdout"),
		)
		require.NoError(t, err)
		assert.Equal(t, os.Stdout, server.OutputWriter)
	})
}

func TestServerStartStop(t *testing.T) {
	t.Run("Start and stop server", func(t *testing.T) {
		server, err := httpserver.New(
			httpserver.WithListenAddr(":0"), // random available port
		)
		require.NoError(t, err)

		// Start server in goroutine
		errChan := make(chan error, 1)
		go func() {
			errChan <- server.Start()
		}()

		// Give server time to start
		time.Sleep(100 * time.Millisecond)

		// Stop server
		err = server.Stop()
		assert.NoError(t, err)

		// Check start returned without error (server closed)
		select {
		case err := <-errChan:
			assert.NoError(t, err)
		case <-time.After(2 * time.Second):
			t.Fatal("server did not stop in time")
		}
	})
}

func TestDebugHandler(t *testing.T) {
	t.Run("GET request", func(t *testing.T) {
		server, err := httpserver.New()
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("User-Agent", "test-agent")
		rec := httptest.NewRecorder()

		server.HTTPServer.Handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "OK\n", rec.Body.String())
	})

	t.Run("POST request with JSON body", func(t *testing.T) {
		server, err := httpserver.New()
		require.NoError(t, err)

		body := `{"name": "test", "value": 123}`
		req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		server.HTTPServer.Handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "OK")
	})

	t.Run("POST request with plain text body", func(t *testing.T) {
		server, err := httpserver.New()
		require.NoError(t, err)

		body := "plain text content"
		req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
		req.Header.Set("Content-Type", "text/plain")
		rec := httptest.NewRecorder()

		server.HTTPServer.Handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("PUT request", func(t *testing.T) {
		server, err := httpserver.New()
		require.NoError(t, err)

		body := `{"update": true}`
		req := httptest.NewRequest(http.MethodPut, "/resource", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		server.HTTPServer.Handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("PATCH request", func(t *testing.T) {
		server, err := httpserver.New()
		require.NoError(t, err)

		body := `{"patch": "data"}`
		req := httptest.NewRequest(http.MethodPatch, "/resource", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		server.HTTPServer.Handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestHMACValidation(t *testing.T) {
	secret := "test-secret"
	headerName := "X-Hub-Signature-256"

	t.Run("Valid HMAC signature", func(t *testing.T) {
		server, err := httpserver.New(
			httpserver.WithHMACSecret(secret),
			httpserver.WithHMACHeaderName(headerName),
		)
		require.NoError(t, err)

		body := `{"action": "test"}`

		// Generate valid signature
		h := hmac.New(sha256.New, []byte(secret))
		h.Write([]byte(body))
		signature := "sha256=" + hex.EncodeToString(h.Sum(nil))

		req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(headerName, signature)
		rec := httptest.NewRecorder()

		server.HTTPServer.Handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("Invalid HMAC signature", func(t *testing.T) {
		server, err := httpserver.New(
			httpserver.WithHMACSecret(secret),
			httpserver.WithHMACHeaderName(headerName),
		)
		require.NoError(t, err)

		body := `{"action": "test"}`
		invalidSignature := "sha256=invalidsignature"

		req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(headerName, invalidSignature)
		rec := httptest.NewRecorder()

		server.HTTPServer.Handler.ServeHTTP(rec, req)

		// Handler still returns OK, but logs the invalid signature
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestSecretTokenValidation(t *testing.T) {
	token := "my-secret-token"
	headerName := "X-Gitlab-Token"

	t.Run("Valid secret token", func(t *testing.T) {
		server, err := httpserver.New(
			httpserver.WithSecretToken(token),
			httpserver.WithSecretTokenHeaderName(headerName),
		)
		require.NoError(t, err)

		body := `{"event": "push"}`
		req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(headerName, token)
		rec := httptest.NewRecorder()

		server.HTTPServer.Handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("Invalid secret token", func(t *testing.T) {
		server, err := httpserver.New(
			httpserver.WithSecretToken(token),
			httpserver.WithSecretTokenHeaderName(headerName),
		)
		require.NoError(t, err)

		body := `{"event": "push"}`
		req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(headerName, "wrong-token")
		rec := httptest.NewRecorder()

		server.HTTPServer.Handler.ServeHTTP(rec, req)

		// Handler still returns OK, but logs the mismatch
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestOutputWriter(t *testing.T) {
	t.Run("Write to temp file", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "httpserver-test-*.log")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		server, err := httpserver.New(
			httpserver.WithOutputWriter(tmpFile.Name()),
		)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		server.HTTPServer.Handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		// Close the output writer to flush
		server.OutputWriter.Close()

		// Read the file and check it has content
		content, err := os.ReadFile(tmpFile.Name())
		require.NoError(t, err)
		assert.NotEmpty(t, content)
		assert.Contains(t, string(content), "Basic HTTP Debugger")
	})
}

func TestInvalidJSONBody(t *testing.T) {
	t.Run("Malformed JSON", func(t *testing.T) {
		server, err := httpserver.New()
		require.NoError(t, err)

		body := `{"invalid json`
		req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		server.HTTPServer.Handler.ServeHTTP(rec, req)

		// Handler returns OK even with invalid JSON
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

// errorReader is a helper that always returns an error when read.
type errorReader struct{}

func (errorReader) Read(_ []byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}

func TestBodyReadError(t *testing.T) {
	t.Run("Error reading body", func(t *testing.T) {
		server, err := httpserver.New()
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/webhook", errorReader{})
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		server.HTTPServer.Handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestMultipleHeaders(t *testing.T) {
	t.Run("Request with multiple headers", func(t *testing.T) {
		server, err := httpserver.New()
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Custom-Header-1", "value1")
		req.Header.Set("X-Custom-Header-2", "value2")
		req.Header.Set("Authorization", "Bearer token")
		req.Header.Set("Accept", "application/json")
		rec := httptest.NewRecorder()

		server.HTTPServer.Handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestSaveRawHTTPRequest(t *testing.T) {
	t.Run("Save raw request to file", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "httpserver-raw-test")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		// Change to temp dir for file saving
		oldWd, _ := os.Getwd()
		os.Chdir(tmpDir)
		defer os.Chdir(oldWd)

		server, err := httpserver.New(
			httpserver.WithSaveRawHTTPRequest(true),
			httpserver.WithRawHTTPRequestFileSaveFormat("test-request.raw"),
		)
		require.NoError(t, err)

		body := `{"test": "data"}`
		req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		server.HTTPServer.Handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		// Check that the raw file was created
		files, err := os.ReadDir(tmpDir)
		require.NoError(t, err)

		found := false
		for _, f := range files {
			if strings.HasSuffix(f.Name(), ".raw") {
				found = true
				content, err := os.ReadFile(tmpDir + "/" + f.Name())
				require.NoError(t, err)
				assert.Contains(t, string(content), "POST /webhook")
				break
			}
		}
		assert.True(t, found, "expected .raw file to be created")
	})
}

func TestVerboseServerInterface(t *testing.T) {
	t.Run("DebugServer implements VerboseServer", func(t *testing.T) {
		server, err := httpserver.New()
		require.NoError(t, err)

		// This test verifies the interface is satisfied
		var _ httpserver.VerboseServer = server
	})
}

func TestLargePayload(t *testing.T) {
	t.Run("Handle large JSON payload", func(t *testing.T) {
		server, err := httpserver.New()
		require.NoError(t, err)

		// Create a large JSON payload
		var buf bytes.Buffer
		buf.WriteString(`{"items": [`)
		for i := 0; i < 100; i++ {
			if i > 0 {
				buf.WriteString(",")
			}
			buf.WriteString(`{"id": ` + string(rune('0'+i%10)) + `, "name": "item"}`)
		}
		buf.WriteString(`]}`)

		req := httptest.NewRequest(http.MethodPost, "/webhook", &buf)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		server.HTTPServer.Handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestFormURLEncodedBody(t *testing.T) {
	t.Run("POST request with form urlencoded body", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "httpserver-form-test-*.log")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		server, err := httpserver.New(
			httpserver.WithOutputWriter(tmpFile.Name()),
		)
		require.NoError(t, err)

		body := "username=john&password=secret123&remember=true"
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()

		server.HTTPServer.Handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		server.OutputWriter.Close()

		content, err := os.ReadFile(tmpFile.Name())
		require.NoError(t, err)
		assert.Contains(t, string(content), "Form Data")
		assert.Contains(t, string(content), "username")
		assert.Contains(t, string(content), "john")
		assert.Contains(t, string(content), "password")
		assert.Contains(t, string(content), "secret123")
		assert.Contains(t, string(content), "remember")
		assert.Contains(t, string(content), "true")
	})

	t.Run("POST request with form urlencoded and charset", func(t *testing.T) {
		server, err := httpserver.New()
		require.NoError(t, err)

		body := "name=Ali+Veli&city=Istanbul"
		req := httptest.NewRequest(http.MethodPost, "/form", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
		rec := httptest.NewRecorder()

		server.HTTPServer.Handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("POST request with URL encoded special characters", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "httpserver-form-special-*.log")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		server, err := httpserver.New(
			httpserver.WithOutputWriter(tmpFile.Name()),
		)
		require.NoError(t, err)

		body := "email=user%40example.com&query=hello+world&special=%26%3D%3F"
		req := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()

		server.HTTPServer.Handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		server.OutputWriter.Close()

		content, err := os.ReadFile(tmpFile.Name())
		require.NoError(t, err)
		assert.Contains(t, string(content), "email")
		assert.Contains(t, string(content), "user@example.com")
		assert.Contains(t, string(content), "query")
		assert.Contains(t, string(content), "hello world")
	})

	t.Run("POST request with empty form", func(t *testing.T) {
		server, err := httpserver.New()
		require.NoError(t, err)

		body := ""
		req := httptest.NewRequest(http.MethodPost, "/empty", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()

		server.HTTPServer.Handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("POST request with multiple values for same key", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "httpserver-form-multi-*.log")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		server, err := httpserver.New(
			httpserver.WithOutputWriter(tmpFile.Name()),
		)
		require.NoError(t, err)

		body := "color=red&color=green&color=blue"
		req := httptest.NewRequest(http.MethodPost, "/colors", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()

		server.HTTPServer.Handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		server.OutputWriter.Close()

		content, err := os.ReadFile(tmpFile.Name())
		require.NoError(t, err)
		assert.Contains(t, string(content), "color")
		assert.Contains(t, string(content), "red, green, blue")
	})

	t.Run("PUT request with form urlencoded body", func(t *testing.T) {
		server, err := httpserver.New()
		require.NoError(t, err)

		body := "status=active&role=admin"
		req := httptest.NewRequest(http.MethodPut, "/user/123", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()

		server.HTTPServer.Handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("Invalid form data", func(t *testing.T) {
		server, err := httpserver.New()
		require.NoError(t, err)

		body := "%invalid%form%data%"
		req := httptest.NewRequest(http.MethodPost, "/invalid", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()

		server.HTTPServer.Handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestMultipartFormData(t *testing.T) {
	t.Run("POST request with multipart form data", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "httpserver-multipart-test-*.log")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		server, err := httpserver.New(
			httpserver.WithOutputWriter(tmpFile.Name()),
		)
		require.NoError(t, err)

		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)

		// Add form field
		_ = writer.WriteField("username", "vigo")
		_ = writer.WriteField("description", "Test upload")

		// Add small text file
		part, _ := writer.CreateFormFile("config", "config.json")
		_, _ = part.Write([]byte(`{"theme": "dark"}`))

		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/upload", &buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rec := httptest.NewRecorder()

		server.HTTPServer.Handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		server.OutputWriter.Close()

		content, err := os.ReadFile(tmpFile.Name())
		require.NoError(t, err)
		assert.Contains(t, string(content), "Form Data")
		assert.Contains(t, string(content), "username")
		assert.Contains(t, string(content), "vigo")
		assert.Contains(t, string(content), "description")
		assert.Contains(t, string(content), "Files")
		assert.Contains(t, string(content), "config.json")
	})

	t.Run("POST request with only file upload", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "httpserver-multipart-file-*.log")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		server, err := httpserver.New(
			httpserver.WithOutputWriter(tmpFile.Name()),
		)
		require.NoError(t, err)

		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)

		// Add a file only
		part, _ := writer.CreateFormFile("avatar", "avatar.png")
		// Write some fake PNG data (larger than 1KB to skip content display)
		fakeData := make([]byte, 2048)
		for i := range fakeData {
			fakeData[i] = byte(i % 256)
		}
		_, _ = part.Write(fakeData)

		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/upload", &buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rec := httptest.NewRecorder()

		server.HTTPServer.Handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		server.OutputWriter.Close()

		content, err := os.ReadFile(tmpFile.Name())
		require.NoError(t, err)
		assert.Contains(t, string(content), "Files")
		assert.Contains(t, string(content), "avatar.png")
		assert.Contains(t, string(content), "2.0 KB")
	})

	t.Run("POST request with small text file shows content", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "httpserver-multipart-small-*.log")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		server, err := httpserver.New(
			httpserver.WithOutputWriter(tmpFile.Name()),
		)
		require.NoError(t, err)

		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)

		// Add small JSON file
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", `form-data; name="data"; filename="data.json"`)
		h.Set("Content-Type", "application/json")
		part, _ := writer.CreatePart(h)
		_, _ = part.Write([]byte(`{"name": "test"}`))

		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/upload", &buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rec := httptest.NewRecorder()

		server.HTTPServer.Handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		server.OutputWriter.Close()

		content, err := os.ReadFile(tmpFile.Name())
		require.NoError(t, err)
		assert.Contains(t, string(content), "data.json")
		assert.Contains(t, string(content), `{"name": "test"}`)
	})

	t.Run("POST request with multiple values for same field", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "httpserver-multipart-multi-*.log")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		server, err := httpserver.New(
			httpserver.WithOutputWriter(tmpFile.Name()),
		)
		require.NoError(t, err)

		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)

		_ = writer.WriteField("tag", "go")
		_ = writer.WriteField("tag", "http")
		_ = writer.WriteField("tag", "debug")

		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/tags", &buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rec := httptest.NewRecorder()

		server.HTTPServer.Handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		server.OutputWriter.Close()

		content, err := os.ReadFile(tmpFile.Name())
		require.NoError(t, err)
		assert.Contains(t, string(content), "tag")
		assert.Contains(t, string(content), "go, http, debug")
	})

	t.Run("Invalid multipart boundary", func(t *testing.T) {
		server, err := httpserver.New()
		require.NoError(t, err)

		body := "invalid multipart data"
		req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader(body))
		req.Header.Set("Content-Type", "multipart/form-data")
		rec := httptest.NewRecorder()

		server.HTTPServer.Handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
