package httpserver

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/vbyazilim/basichttpdebugger/internal/release"
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
		d.OutputWriter = os.Stdout

		if s != "stdout" {
			fwriter, err := os.Create(filepath.Clean(s))
			if err == nil {
				d.OutputWriter = fwriter
			} else {
				d.OutputWriter = nil
			}
		}
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

type debugHandlerOptions struct {
	writer                       io.WriteCloser
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
				expectedSignature := "sha256=" + hex.EncodeToString(h.Sum(nil))

				t.AppendRows([]table.Row{
					{"Incoming Signature", signature},
					{"Expected Signature", expectedSignature},
					{"Is Valid?", hmac.Equal([]byte(expectedSignature), []byte(signature))},
				})
				t.AppendSeparator()
			}
			requestContentType := r.Header.Get("Content-Type")
			t.AppendRow(table.Row{"Incoming", requestContentType})
			t.AppendSeparator()

			bodyAsString = string(body)

			switch requestContentType {
			case "application/json":
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
			fmt.Fprintf(mwr, "\n%s\n", bodyAsString)
		}
		options.drawLine()
		if rawHRw != nil {
			_ = rawHRw.Close()
		}
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
