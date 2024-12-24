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
	"github.com/vbyazilim/basichttpdebugger/internal/validateutils"
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
	HTTPServer        *http.Server
	OutputWriter      io.WriteCloser
	ListenAddr        string
	HMACSecret        string
	HMACHeaderName    string
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	Color             bool
}

// Start starts http server.
func (s *DebugServer) Start() error {
	log.Printf("server listening at %s\n", s.ListenAddr)
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

// WithColor enables/disables colorful output.
func WithColor(b bool) Option {
	return func(d *DebugServer) {
		d.Color = b
	}
}

type debugHandlerOptions struct {
	writer         io.WriteCloser
	hmacSecret     string
	hmacHeaderName string
	color          bool
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
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")

		now := time.Now().UTC()

		options.drawLine()

		t := table.NewWriter()
		t.SetOutputMirror(options.writer)
		t.SetTitle("Basic HTTP Debugger")
		t.SetAllowedRowLength(options.getTerminalWidth())
		t.AppendRows([]table.Row{
			{"Version", release.Version},
			{"Build", release.BuildInformation[:12]},
			{"Request Time", now},
			{"HTTP Method", r.Method},
		})
		t.AppendSeparator()
		t.AppendRow(table.Row{"Request Headers", "Request Headers"}, table.RowConfig{
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

		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch:
			t.AppendSeparator()
			t.AppendRow(table.Row{"Payload", "Payload"}, table.RowConfig{
				AutoMerge:      true,
				AutoMergeAlign: text.AlignLeft,
			})
			t.AppendSeparator()

			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.AppendRow(table.Row{err.Error(), err.Error()}, table.RowConfig{
					AutoMerge:      true,
					AutoMergeAlign: text.AlignLeft,
				})
				t.AppendSeparator()

				goto RENDER
			}
			defer func() { _ = r.Body.Close() }()

			if options.hmacSecret != "" {
				t.AppendRow(table.Row{"HMAC Secret", options.hmacHeaderName})
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

			switch requestContentType {
			case "application/json":
				var jsonBody map[string]any
				if err = json.Unmarshal(body, &jsonBody); err != nil {
					t.AppendRow(table.Row{err.Error(), err.Error()}, table.RowConfig{
						AutoMerge:      true,
						AutoMergeAlign: text.AlignLeft,
					})
					t.AppendSeparator()

					goto RENDER
				}

				prettyJSON, errpj := json.MarshalIndent(jsonBody, "", "    ")
				if errpj != nil {
					t.AppendRow(table.Row{errpj.Error(), errpj.Error()}, table.RowConfig{
						AutoMerge:      true,
						AutoMergeAlign: text.AlignLeft,
					})
					t.AppendSeparator()

					goto RENDER
				}

				t.AppendSeparator()
				t.AppendRow(table.Row{string(prettyJSON), string(prettyJSON)}, table.RowConfig{
					AutoMerge:      true,
					AutoMergeAlign: text.AlignLeft,
				})
			default:
				t.AppendSeparator()
				t.AppendRow(table.Row{string(body), string(body)}, table.RowConfig{
					AutoMerge:      true,
					AutoMergeAlign: text.AlignLeft,
				})
			}
		}
	RENDER:
		t.Render()

		// fmt.Fprintf(options.writer, "Time: %s\n", now)
		// fmt.Println(options)
		// fmt.Println(r.Method)
		// fmt.Println(options.getTerminalWidth())
		options.drawLine()
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

	handlerOptions := debugHandlerOptions{
		writer:         opts.OutputWriter,
		hmacSecret:     opts.HMACSecret,
		hmacHeaderName: opts.HMACHeaderName,
		color:          opts.Color,
	}

	fmt.Printf("opts: %+v\n", opts)
	fmt.Printf("handlerOptions: %+v\n", handlerOptions)

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
