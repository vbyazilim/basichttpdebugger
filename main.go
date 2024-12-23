package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/vbyazilim/basichttpdebugger/release"
	"github.com/vigo/accept"
	"golang.org/x/term"
)

const (
	defReadTimeout       = 5 * time.Second
	defReadHeaderTimeout = 5 * time.Second
	defWriteTimeout      = 10 * time.Second
	defIdleTimeout       = 15 * time.Second
	defTerminalWidth     = 120
)

var (
	defHMACSecret     string
	defHMACHeaderName = "X-Hub-Signature-256"
	defListenAddr     = ":9002"
	defOutput         = "stdout"
	defColor          bool
)

func main() {
	if val := os.Getenv("HMAC_SECRET"); val != "" {
		defHMACSecret = val
	}
	if val := os.Getenv("HMAC_HEADER_NAME"); val != "" {
		defHMACHeaderName = val
	}
	if val := os.Getenv("HOST"); val != "" {
		defListenAddr = val
	}
	if val := os.Getenv("OUTPUT"); val != "" {
		defOutput = val
	}
	if val := os.Getenv("COLOR"); val != "" {
		defColor = true
	}

	hmacSecretValue := flag.String("hmac-secret", defHMACSecret, "your HMAC secret value")
	hmacHeaderName := flag.String("hmac-header-name", defHMACHeaderName, "name of your signature header")
	listenAddr := flag.String("listen", defListenAddr, "listen addr")
	color := flag.Bool("color", defColor, "enable color")
	output := flag.String("output", defOutput, "output to")
	flag.Parse()

	if *color {
		text.EnableColors()
	} else {
		text.DisableColors()
	}

	var outputWriter *os.File

	if *output == "stdout" {
		outputWriter = os.Stdout
	} else {
		fileWriter, err := os.Create(*output)
		if err != nil {
			log.Fatal("can not create file", err)
		}

		outputWriter = fileWriter

		defer func() { _ = outputWriter.Close() }()
	}

	cn := accept.New(
		accept.WithSupportedMediaTypes("text/plain"),
		accept.WithDefaultMediaType("text/plain"),
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/", debugHandlerFunc(cn, outputWriter, *hmacSecretValue, *hmacHeaderName))

	server := &http.Server{
		Addr:              *listenAddr,
		Handler:           mux,
		ReadTimeout:       defReadTimeout,
		ReadHeaderTimeout: defReadHeaderTimeout,
		WriteTimeout:      defWriteTimeout,
		IdleTimeout:       defIdleTimeout,
	}

	log.Printf("server listening at %s\n", *listenAddr)
	if errsrv := server.ListenAndServe(); errsrv != nil && !errors.Is(errsrv, http.ErrServerClosed) {
		log.Printf("server error: %v\n", errsrv)
	}
}

func getTerminalWidth() int {
	if width, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
		return width
	}

	return defTerminalWidth
}

func errorAsTable(fwriter *os.File, title string, err error) {
	t := table.NewWriter()
	t.SetOutputMirror(fwriter)
	t.SetTitle(text.Colors{text.Bold, text.FgRed}.Sprint(title))
	t.AppendRow(table.Row{err.Error()})
	t.Render()
}

func debugHandlerFunc(cn *accept.ContentNegotiation, fwriter *os.File, hmsv string, hmhn string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		terminalWidth := getTerminalWidth()

		acceptHeader := r.Header.Get("Accept")
		requestContentType := r.Header.Get("Content-Type")
		contentType := cn.Negotiate(acceptHeader)

		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")

		mainTitle := "Basic HTTP Debugger - v" + release.Version + release.BuildInformation

		fmt.Fprintln(fwriter, strings.Repeat("-", terminalWidth))
		t := table.NewWriter()
		t.SetOutputMirror(fwriter)
		t.SetTitle(text.Colors{text.Bold, text.FgWhite}.Sprint(mainTitle))
		t.SetColumnConfigs([]table.ColumnConfig{
			{Number: 1, Colors: text.Colors{text.FgYellow}},
		})

		infoRows := []table.Row{
			{"HTTP Method", r.Method},
			{"Matching Content-Type", contentType},
		}
		t.AppendRows(infoRows)
		t.Render()

		t = table.NewWriter()
		t.SetOutputMirror(fwriter)
		t.SetTitle(text.Colors{text.Bold, text.FgWhite}.Sprint("Request Headers"))

		t.SetColumnConfigs([]table.ColumnConfig{
			{Number: 1, Colors: text.Colors{text.FgYellow}},
			{Number: 2, WidthMax: (terminalWidth / 2) - 2},
		})

		headerKeys := make([]string, 0, len(r.Header))
		for key := range r.Header {
			headerKeys = append(headerKeys, key)
		}
		sort.Strings(headerKeys)

		for _, key := range headerKeys {
			t.AppendRow(table.Row{key, strings.Join(r.Header[key], ",")})
		}
		t.Render()

		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch:
			body, err := io.ReadAll(r.Body)
			if err != nil {
				errorAsTable(fwriter, "Body READ error", err)

				return
			}
			defer func() { _ = r.Body.Close() }()

			t = table.NewWriter()
			t.SetOutputMirror(fwriter)
			t.SetTitle(text.Colors{text.Bold, text.FgWhite}.Sprint("HMAC Validation"))
			t.SetColumnConfigs([]table.ColumnConfig{
				{Number: 1, Colors: text.Colors{text.FgYellow}},
			})
			if hmsv != "" {
				t.AppendRow(table.Row{"HMAC Secret Value", hmsv})
			}
			if hmhn != "" {
				t.AppendRow(table.Row{"HMAC Header Name", hmhn})
			}

			if hmsv != "" && hmhn != "" {
				signature := r.Header.Get(hmhn)
				h := hmac.New(sha256.New, []byte(hmsv))
				_, _ = h.Write(body)
				expectedSignature := "sha256=" + hex.EncodeToString(h.Sum(nil))

				t.AppendRows([]table.Row{
					{"Incoming Signature", signature},
					{"Expected Signature", expectedSignature},
					{"Is Valid?", hmac.Equal([]byte(expectedSignature), []byte(signature))},
				})
			}
			t.Render()

			switch requestContentType {
			case "application/json":
				var jsonBody map[string]any
				if err = json.Unmarshal(body, &jsonBody); err != nil {
					errorAsTable(fwriter, "json.Unmarshal error", err)

					return
				}

				prettyJSON, errpj := json.MarshalIndent(jsonBody, "", "    ")
				if errpj != nil {
					errorAsTable(fwriter, "json.MarshalIndent error", errpj)

					return
				}
				fmt.Fprintln(fwriter, string(prettyJSON))
			default:
				fmt.Fprintln(fwriter, string(body))
			}

			fmt.Fprintln(fwriter, strings.Repeat("-", terminalWidth))
		}
	}
}
