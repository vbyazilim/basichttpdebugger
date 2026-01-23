package httpserver

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/vbyazilim/basichttpdebugger/internal/release"
	"github.com/vbyazilim/basichttpdebugger/internal/requeststore"
	"github.com/vbyazilim/basichttpdebugger/internal/stringutils"
	"github.com/vbyazilim/basichttpdebugger/internal/validateutils"
	"github.com/vbyazilim/basichttpdebugger/internal/writerutils"
	"golang.org/x/term"
)

var _ VerboseServer = (*DebugServer)(nil) // compile time proof

// sentinel errors.
var (
	ErrValueRequired = errors.New("value required")
)

const (
	defReadTimeout       = 5 * time.Second
	defReadHeaderTimeout = 5 * time.Second
	defWriteTimeout      = 10 * time.Second
	defIdleTimeout       = 15 * time.Second
	defListenAddr        = ":9002"
	defTerminalWidth     = 80

	headerContentType   = "Content-Type"
	asciiSpaceThreshold = 32      // ASCII control characters below this are non-printable
	maxImagePreviewSize = 5 << 20 // 5MB max for image preview in WebUI
)

// VerboseServer defines server behaviours.
type VerboseServer interface {
	Start() error
	Stop() error
}

// DebugServer represents server/handler args.
type DebugServer struct {
	HTTPServer                   *http.Server
	OutputWriter                 io.WriteCloser
	Store                        *requeststore.Store
	ListenAddr                   string
	HMACSecret                   string
	HMACHeaderName               string
	RawHTTPRequestFileSaveFormat string
	SecretToken                  string
	SecretTokenHeaderName        string
	ReadTimeout                  time.Duration
	ReadHeaderTimeout            time.Duration
	WriteTimeout                 time.Duration
	IdleTimeout                  time.Duration
	Color                        bool
	SaveRawHTTPRequest           bool
}

// Start starts http server.
func (s *DebugServer) Start() error {
	log.Printf("server listening at %s\n", s.ListenAddr)
	if s.HMACSecret != "" {
		log.Print("hmac-secret et")
	}
	if s.HMACHeaderName != "" {
		log.Printf("hmac-header-name: %s\n", s.HMACHeaderName)
	}

	if fname := writerutils.GetFilePathName(s.OutputWriter); fname != "" {
		log.Printf("output is set to %s\n", fname)
	}
	if s.SaveRawHTTPRequest {
		log.Println("saving raw http request is enabled")
	}
	if err := s.HTTPServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server start error: %w", err)
	}

	return nil
}

// Stop stops/shutdowns server.
func (s *DebugServer) Stop() error {
	if err := s.HTTPServer.Shutdown(context.Background()); err != nil {
		return fmt.Errorf("server stop error: %w", err)
	}

	return nil
}

// Option represents option function type.
type Option func(*DebugServer)

// WithListenAddr sets listen addr.
func WithListenAddr(addr string) Option {
	return func(d *DebugServer) {
		d.ListenAddr = addr
	}
}

// WithReadTimeout sets http server's read timeout.
func WithReadTimeout(dur time.Duration) Option {
	return func(d *DebugServer) {
		d.ReadTimeout = dur
	}
}

// WithReadHeaderTimeout sets http server's read header timeout.
func WithReadHeaderTimeout(dur time.Duration) Option {
	return func(d *DebugServer) {
		d.ReadHeaderTimeout = dur
	}
}

// WithWriteTimeout sets http server's write timeout.
func WithWriteTimeout(dur time.Duration) Option {
	return func(d *DebugServer) {
		d.WriteTimeout = dur
	}
}

// WithIdleTimeout sets http server's idle timeout.
func WithIdleTimeout(dur time.Duration) Option {
	return func(d *DebugServer) {
		d.IdleTimeout = dur
	}
}

// WithOutputWriter sets output, where to write incoming webhook.
func WithOutputWriter(s string) Option {
	return func(d *DebugServer) {
		if s == "stdout" {
			d.OutputWriter = os.Stdout
			return
		}

		fwriter, err := os.Create(filepath.Clean(s))
		if err != nil {
			d.OutputWriter = nil
			return
		}
		d.OutputWriter = fwriter
	}
}

// WithHMACSecret sets HMAC secret value.
func WithHMACSecret(s string) Option {
	return func(d *DebugServer) {
		d.HMACSecret = s
	}
}

// WithHMACHeaderName sets HMAC header name value, will check this http header
// name in request header.
func WithHMACHeaderName(s string) Option {
	return func(d *DebugServer) {
		d.HMACHeaderName = s
	}
}

// WithSecretToken sets the secret value for secret token.
func WithSecretToken(s string) Option {
	return func(d *DebugServer) {
		d.SecretToken = s
	}
}

// WithSecretTokenHeaderName sets secret token header name value, will check this
// http header name in request header.
func WithSecretTokenHeaderName(s string) Option {
	return func(d *DebugServer) {
		d.SecretTokenHeaderName = s
	}
}

// WithColor enables/disables colorful output.
func WithColor(b bool) Option {
	return func(d *DebugServer) {
		d.Color = b
	}
}

// WithSaveRawHTTPRequest enables/disables saving raw http request to disk.
func WithSaveRawHTTPRequest(b bool) Option {
	return func(d *DebugServer) {
		d.SaveRawHTTPRequest = b
	}
}

// WithRawHTTPRequestFileSaveFormat set file save name format for raw http request.
func WithRawHTTPRequestFileSaveFormat(s string) Option {
	return func(d *DebugServer) {
		d.RawHTTPRequestFileSaveFormat = s
	}
}

// WithStore sets the request store for web dashboard.
func WithStore(s *requeststore.Store) Option {
	return func(d *DebugServer) {
		d.Store = s
	}
}

type debugHandlerOptions struct {
	writer                       io.WriteCloser
	store                        *requeststore.Store
	hmacSecret                   string
	hmacHeaderName               string
	secretToken                  string
	secretTokenHeaderName        string
	rawHTTPRequestFileSaveFormat string
	color                        bool
	saveRawHTTPRequest           bool
}

func (debugHandlerOptions) getTerminalWidth() int {
	if width, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
		return width
	}

	return defTerminalWidth
}

func (dh debugHandlerOptions) drawLine() {
	fmt.Fprintln(dh.writer, strings.Repeat("-", dh.getTerminalWidth()))
}

func debugHandlerFunc(options *debugHandlerOptions) http.HandlerFunc {
	colorTitle := text.Colors{text.Bold, text.FgWhite}
	colorPayload := text.Colors{text.FgCyan}
	colorError := text.Colors{text.BlinkSlow, text.FgRed}

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")

		now := time.Now().UTC()

		options.drawLine()

		t := table.NewWriter()
		t.SetOutputMirror(options.writer)
		t.SetTitle(colorTitle.Sprint("Basic HTTP Debugger"))

		filename := writerutils.GetFilePathName(options.writer)
		if filename == "/dev/stdout" {
			t.SetAllowedRowLength(options.getTerminalWidth())
		} else {
			fmt.Fprintln(w, "to see the result, run")
			fmt.Fprintf(w, "tail -f %s\n", filename)
		}
		t.SetColumnConfigs([]table.ColumnConfig{
			{Number: 1, Colors: text.Colors{text.FgYellow}},
		})
		t.AppendRows([]table.Row{
			{"Version", release.Version},
			{"Build", release.BuildInformation[:12]},
			{"Request Time", now},
			{"HTTP Method", r.Method},
		})
		t.AppendSeparator()
		titleRequestHeaders := colorTitle.Sprint("Request Headers")
		t.AppendRow(table.Row{titleRequestHeaders, titleRequestHeaders}, table.RowConfig{
			AutoMerge:      true,
			AutoMergeAlign: text.AlignLeft,
		})
		t.AppendSeparator()

		headerKeys := make([]string, 0, len(r.Header))
		for key := range r.Header {
			headerKeys = append(headerKeys, key)
		}
		sort.Strings(headerKeys)

		for _, key := range headerKeys {
			t.AppendRow(table.Row{key, strings.Join(r.Header[key], ",")})
		}

		var bodyAsString string
		var storeFiles []requeststore.FileAttachment

		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch:
			t.AppendSeparator()
			titlePayload := colorTitle.Sprint("Payload")
			t.AppendRow(table.Row{titlePayload, titlePayload}, table.RowConfig{
				AutoMerge:      true,
				AutoMergeAlign: text.AlignLeft,
			})
			t.AppendSeparator()

			body, err := io.ReadAll(r.Body)
			if err != nil {
				txtErrorRead := colorError.Sprintf("read error: %s", err.Error())
				t.AppendRow(table.Row{txtErrorRead, txtErrorRead}, table.RowConfig{
					AutoMerge:      true,
					AutoMergeAlign: text.AlignLeft,
				})
				t.AppendSeparator()

				goto RENDER
			}
			defer func() { _ = r.Body.Close() }()

			if options.secretToken != "" {
				t.AppendRow(table.Row{"Secret Token", options.secretToken})
			}
			if options.secretTokenHeaderName != "" {
				t.AppendRow(table.Row{"Secret Token Header Name", options.secretTokenHeaderName})
			}

			if options.secretToken != "" && options.secretTokenHeaderName != "" {
				t.AppendRows([]table.Row{
					{"Secret Token Matches?", r.Header.Get(options.secretTokenHeaderName) == options.secretToken},
				})
				t.AppendSeparator()
			}

			if options.hmacSecret != "" {
				t.AppendRow(table.Row{"HMAC Secret", options.hmacSecret})
			}
			if options.hmacHeaderName != "" {
				t.AppendRow(table.Row{"HMAC Header Name", options.hmacHeaderName})
			}
			if options.hmacSecret != "" && options.hmacHeaderName != "" {
				signature := r.Header.Get(options.hmacHeaderName)
				h := hmac.New(sha256.New, []byte(options.hmacSecret))
				_, _ = h.Write(body)

				computedHash := hex.EncodeToString(h.Sum(nil))
				cleanSignature := strings.TrimPrefix(signature, "sha256=")

				t.AppendRows([]table.Row{
					{"Incoming Signature", cleanSignature},
					{"Expected Signature", computedHash},
					{"Is Valid?", hmac.Equal([]byte(computedHash), []byte(cleanSignature))},
				})
				t.AppendSeparator()
			}
			requestContentType := r.Header.Get("Content-Type")
			t.AppendRow(table.Row{"Incoming", requestContentType})
			t.AppendSeparator()

			bodyAsString = string(body)

			switch {
			case requestContentType == "application/json":
				var jsonBody map[string]any
				if err = json.Unmarshal(body, &jsonBody); err != nil {
					txtErrorUnmarshal := colorError.Sprintf("json.Unmarshal error: %s", err.Error())
					t.AppendRow(table.Row{txtErrorUnmarshal, txtErrorUnmarshal}, table.RowConfig{
						AutoMerge:      true,
						AutoMergeAlign: text.AlignLeft,
					})
					t.AppendSeparator()

					goto RENDER
				}

				prettyJSON, errpj := json.MarshalIndent(jsonBody, "", "    ")
				if errpj != nil {
					txtErrorMarshalIndent := colorError.Sprintf("json.MarshalIndent error: %s", errpj.Error())
					t.AppendRow(table.Row{txtErrorMarshalIndent, txtErrorMarshalIndent}, table.RowConfig{
						AutoMerge:      true,
						AutoMergeAlign: text.AlignLeft,
					})
					t.AppendSeparator()

					goto RENDER
				}

				t.AppendSeparator()
				payloadJSON := colorPayload.Sprintf("%s", prettyJSON)
				t.AppendRow(table.Row{payloadJSON, payloadJSON}, table.RowConfig{
					AutoMerge:      true,
					AutoMergeAlign: text.AlignLeft,
				})
			case strings.HasPrefix(requestContentType, "application/x-www-form-urlencoded"):
				formData, errForm := url.ParseQuery(bodyAsString)
				if errForm != nil {
					txtErrorForm := colorError.Sprintf("url.ParseQuery error: %s", errForm.Error())
					t.AppendRow(table.Row{txtErrorForm, txtErrorForm}, table.RowConfig{
						AutoMerge:      true,
						AutoMergeAlign: text.AlignLeft,
					})
					t.AppendSeparator()

					goto RENDER
				}

				titleFormData := colorTitle.Sprint("Form Data")
				t.AppendRow(table.Row{titleFormData, titleFormData}, table.RowConfig{
					AutoMerge:      true,
					AutoMergeAlign: text.AlignLeft,
				})
				t.AppendSeparator()

				formKeys := make([]string, 0, len(formData))
				for key := range formData {
					formKeys = append(formKeys, key)
				}
				sort.Strings(formKeys)

				for _, key := range formKeys {
					values := formData[key]
					valueStr := colorPayload.Sprint(strings.Join(values, ", "))
					t.AppendRow(table.Row{key, valueStr})
				}
			case strings.HasPrefix(requestContentType, "multipart/form-data"):
				_, params, errMedia := mime.ParseMediaType(requestContentType)
				if errMedia != nil {
					txtErrorMedia := colorError.Sprintf("mime.ParseMediaType error: %s", errMedia.Error())
					t.AppendRow(table.Row{txtErrorMedia, txtErrorMedia}, table.RowConfig{
						AutoMerge:      true,
						AutoMergeAlign: text.AlignLeft,
					})
					t.AppendSeparator()

					goto RENDER
				}

				boundary := params["boundary"]
				if boundary == "" {
					txtErrorBoundary := colorError.Sprint("multipart boundary not found")
					t.AppendRow(table.Row{txtErrorBoundary, txtErrorBoundary}, table.RowConfig{
						AutoMerge:      true,
						AutoMergeAlign: text.AlignLeft,
					})
					t.AppendSeparator()

					goto RENDER
				}

				reader := multipart.NewReader(bytes.NewReader(body), boundary)

				formFields := make(map[string][]string)
				type fileInfo struct {
					FieldName   string
					Filename    string
					Size        int
					ContentType string
					Content     string
					RawData     []byte // for images
				}
				var files []fileInfo

				const maxContentDisplay = 1024 // 1KB

				for {
					part, errPart := reader.NextPart()
					if errPart == io.EOF {
						break
					}
					if errPart != nil {
						txtErrorPart := colorError.Sprintf("multipart read error: %s", errPart.Error())
						t.AppendRow(table.Row{txtErrorPart, txtErrorPart}, table.RowConfig{
							AutoMerge:      true,
							AutoMergeAlign: text.AlignLeft,
						})

						break
					}

					if part.FileName() == "" {
						// Regular form field - read all (typically small)
						fieldData, _ := io.ReadAll(part)
						_ = part.Close()
						fieldName := part.FormName()
						formFields[fieldName] = append(formFields[fieldName], string(fieldData))
					} else {
						// File upload - stream and limit what we keep in memory
						fi := fileInfo{
							FieldName:   part.FormName(),
							Filename:    part.FileName(),
							ContentType: part.Header.Get(headerContentType),
						}

						isImage := isImageContentType(fi.ContentType)
						isText := isTextContentType(fi.ContentType)

						// Determine buffer limit based on content type
						var bufferLimit int64
						switch {
						case isImage:
							bufferLimit = maxImagePreviewSize
						case isText:
							bufferLimit = maxContentDisplay
						default:
							bufferLimit = 0 // Don't buffer binary non-images
						}

						// Read only up to buffer limit
						var previewData []byte
						if bufferLimit > 0 {
							limitReader := io.LimitReader(part, bufferLimit)
							previewData, _ = io.ReadAll(limitReader)
						}

						// Discard remaining bytes while counting total size
						remainingBytes, _ := io.Copy(io.Discard, part)
						_ = part.Close()

						fi.Size = len(previewData) + int(remainingBytes)

						// Store preview data only if file is within limits
						if isText && fi.Size <= maxContentDisplay {
							fi.Content = string(previewData)
						}
						if isImage && fi.Size <= maxImagePreviewSize {
							fi.RawData = previewData
						}

						files = append(files, fi)
					}
				}

				// Display form fields
				if len(formFields) > 0 {
					titleFormData := colorTitle.Sprint("Form Data")
					t.AppendRow(table.Row{titleFormData, titleFormData}, table.RowConfig{
						AutoMerge:      true,
						AutoMergeAlign: text.AlignLeft,
					})
					t.AppendSeparator()

					fieldKeys := make([]string, 0, len(formFields))
					for key := range formFields {
						fieldKeys = append(fieldKeys, key)
					}
					sort.Strings(fieldKeys)

					for _, key := range fieldKeys {
						values := formFields[key]
						valueStr := colorPayload.Sprint(strings.Join(values, ", "))
						t.AppendRow(table.Row{key, valueStr})
					}
					t.AppendSeparator()
				}

				// Display files
				if len(files) > 0 { //nolint:revive // early-return not applicable in switch case
					titleFiles := colorTitle.Sprint("Files")
					t.AppendRow(table.Row{titleFiles, titleFiles}, table.RowConfig{
						AutoMerge:      true,
						AutoMergeAlign: text.AlignLeft,
					})
					t.AppendSeparator()

					for _, fi := range files {
						sizeStr := formatFileSize(fi.Size)
						fileRow := colorPayload.Sprintf("%s | %s | %s", fi.Filename, sizeStr, fi.ContentType)
						t.AppendRow(table.Row{fileRow, fileRow}, table.RowConfig{
							AutoMerge:      true,
							AutoMergeAlign: text.AlignLeft,
						})

						if fi.Content != "" {
							contentStr := colorPayload.Sprint(fi.Content)
							t.AppendRow(table.Row{contentStr, contentStr}, table.RowConfig{
								AutoMerge:      true,
								AutoMergeAlign: text.AlignLeft,
							})
						}
					}
				}

				// Prepare files for store (with base64 for images)
				for _, fi := range files {
					sf := requeststore.FileAttachment{
						FieldName:   fi.FieldName,
						Filename:    fi.Filename,
						ContentType: fi.ContentType,
						Size:        fi.Size,
					}
					if len(fi.RawData) > 0 {
						sf.Data = base64.StdEncoding.EncodeToString(fi.RawData)
					}
					storeFiles = append(storeFiles, sf)
				}
			default:
				payloadText := colorPayload.Sprintf("%s", body)
				t.AppendSeparator()
				t.AppendRow(
					table.Row{payloadText, payloadText},
					table.RowConfig{
						AutoMerge:      true,
						AutoMergeAlign: text.AlignLeft,
					},
				)
			}
		}
	RENDER:
		t.Render()

		mwr := io.MultiWriter(options.writer)
		var rawHRw *os.File
		var rawErr error

		if options.saveRawHTTPRequest {
			formattedFilename := stringutils.GetFormattedFilename(options.rawHTTPRequestFileSaveFormat, r)
			rawHRw, rawErr = os.Create(filepath.Clean(formattedFilename))
			if rawErr != nil {
				fmt.Println("err", rawErr)

				goto WRITERHR
			}

			mwr = io.MultiWriter(options.writer, rawHRw)
			fmt.Fprintf(w, "Raw HTTP Request is saved to: %s\n", formattedFilename)
		}

	WRITERHR:
		options.drawLine()
		fmt.Fprintln(options.writer, "Raw Http Request")
		options.drawLine()
		fmt.Fprintf(mwr, "%s %s %s\n", r.Method, r.URL.String(), r.Proto)
		fmt.Fprintf(mwr, "Host: %s\n", r.Host)
		for _, key := range headerKeys {
			fmt.Fprintf(mwr, "%s: %s\n", key, strings.Join(r.Header[key], ","))
		}
		if bodyAsString != "" {
			sanitizedBody := sanitizeBodyForDisplay(bodyAsString, r.Header.Get(headerContentType))
			fmt.Fprintf(mwr, "\n%s\n", sanitizedBody)
		}
		options.drawLine()
		if rawHRw != nil {
			_ = rawHRw.Close()
		}

		if options.store == nil {
			return
		}

		headers := make(map[string]string)
		for _, key := range headerKeys {
			headers[key] = strings.Join(r.Header[key], ",")
		}
		options.store.Add(requeststore.Request{
			Time:    now,
			Method:  r.Method,
			URL:     r.URL.String(),
			Headers: headers,
			Body:    bodyAsString,
			Host:    r.Host,
			Proto:   r.Proto,
			Files:   storeFiles,
		})
	}
}

// New instantiates new http server instance.
func New(options ...Option) (*DebugServer, error) {
	opts := &DebugServer{
		ListenAddr:        defListenAddr,
		ReadTimeout:       defReadTimeout,
		ReadHeaderTimeout: defReadHeaderTimeout,
		WriteTimeout:      defWriteTimeout,
		IdleTimeout:       defIdleTimeout,
		OutputWriter:      os.Stdout,
	}

	for _, opt := range options {
		opt(opts)
	}

	if err := validateutils.ValidateNetworkAddress(opts.ListenAddr); err != nil {
		return nil, fmt.Errorf("error listen addr, %w", err)
	}

	if opts.OutputWriter == nil {
		return nil, fmt.Errorf("invalid output: %w", ErrValueRequired)
	}

	targetFilename := writerutils.GetFilePathName(opts.OutputWriter)
	if opts.Color && targetFilename == "/dev/stdout" {
		log.Println("color is enabled")
		text.EnableColors()
	} else {
		text.DisableColors()
	}

	handlerOptions := debugHandlerOptions{
		writer:                       opts.OutputWriter,
		store:                        opts.Store,
		hmacSecret:                   opts.HMACSecret,
		hmacHeaderName:               opts.HMACHeaderName,
		secretToken:                  opts.SecretToken,
		secretTokenHeaderName:        opts.SecretTokenHeaderName,
		color:                        opts.Color,
		rawHTTPRequestFileSaveFormat: opts.RawHTTPRequestFileSaveFormat,
		saveRawHTTPRequest:           opts.SaveRawHTTPRequest,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", debugHandlerFunc(&handlerOptions))

	server := &http.Server{
		Addr:              opts.ListenAddr,
		Handler:           mux,
		ReadTimeout:       opts.ReadTimeout,
		ReadHeaderTimeout: opts.ReadHeaderTimeout,
		WriteTimeout:      opts.WriteTimeout,
		IdleTimeout:       opts.IdleTimeout,
	}

	opts.HTTPServer = server

	return opts, nil
}

// isImageContentType checks if the content type is an image.
func isImageContentType(contentType string) bool {
	return strings.HasPrefix(strings.ToLower(contentType), "image/")
}

// isTextContentType checks if the content type is text-based.
func isTextContentType(contentType string) bool {
	textTypes := []string{
		"text/",
		"application/json",
		"application/xml",
		"application/javascript",
		"application/x-www-form-urlencoded",
	}

	ct := strings.ToLower(contentType)
	for _, t := range textTypes {
		if strings.HasPrefix(ct, t) {
			return true
		}
	}

	return false
}

// formatFileSize formats file size in human readable format.
func formatFileSize(size int) string {
	const (
		kb = 1024
		mb = kb * 1024
	)

	switch {
	case size >= mb:
		return fmt.Sprintf("%.1f MB", float64(size)/float64(mb))
	case size >= kb:
		return fmt.Sprintf("%.1f KB", float64(size)/float64(kb))
	default:
		return fmt.Sprintf("%d B", size)
	}
}

// containsBinaryData checks if the string contains binary (non-printable) characters.
func containsBinaryData(s string) bool {
	for _, r := range s {
		if r < asciiSpaceThreshold && r != '\t' && r != '\n' && r != '\r' {
			return true
		}
	}
	return false
}

// sanitizeBodyForDisplay sanitizes the body for raw HTTP display.
// For multipart requests with binary files, it replaces binary content with placeholders.
func sanitizeBodyForDisplay(body, contentType string) string {
	if body == "" {
		return body
	}

	// For non-multipart requests, check for binary and show placeholder
	if !strings.Contains(contentType, "multipart/form-data") {
		if containsBinaryData(body) {
			return fmt.Sprintf("[binary data: %s]", formatFileSize(len(body)))
		}
		return body
	}

	// For multipart, parse and sanitize each part
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil || !strings.HasPrefix(mediaType, "multipart/") {
		if containsBinaryData(body) {
			return fmt.Sprintf("[binary data: %s]", formatFileSize(len(body)))
		}
		return body
	}

	boundary := params["boundary"]
	if boundary == "" {
		return body
	}

	var result strings.Builder
	parts := strings.Split(body, "--"+boundary)

	for i, part := range parts {
		if strings.TrimSpace(part) == "" || strings.TrimSpace(part) == "--" {
			if i == 0 {
				continue
			}
			result.WriteString("--" + boundary + part)
			continue
		}

		// Find the header/content separator
		headerEnd := strings.Index(part, "\r\n\r\n")
		separator := "\r\n\r\n"
		if headerEnd == -1 {
			headerEnd = strings.Index(part, "\n\n")
			separator = "\n\n"
		}

		if headerEnd == -1 {
			result.WriteString("--" + boundary + part)
			continue
		}

		headers := part[:headerEnd]
		content := part[headerEnd+len(separator):]

		// Check if this part has a filename (it's a file upload)
		hasFilename := strings.Contains(headers, "filename=")

		result.WriteString("--" + boundary)
		result.WriteString(headers)
		result.WriteString(separator)

		if hasFilename && containsBinaryData(content) {
			// Remove trailing boundary markers for size calculation
			cleanContent := strings.TrimSuffix(content, "\r\n")
			cleanContent = strings.TrimSuffix(cleanContent, "\n")
			result.WriteString(fmt.Sprintf("[binary data: %s]", formatFileSize(len(cleanContent))))
			// Preserve the trailing newlines
			if strings.HasSuffix(content, "\r\n") {
				result.WriteString("\r\n")
			} else if strings.HasSuffix(content, "\n") {
				result.WriteString("\n")
			}
		} else {
			result.WriteString(content)
		}
	}

	return result.String()
}
