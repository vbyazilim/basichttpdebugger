package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/vigo/accept"
)

const (
	defReadTimeout       = 5 * time.Second
	defReadHeaderTimeout = 5 * time.Second
	defWriteTimeout      = 10 * time.Second
	defIdleTimeout       = 15 * time.Second
)

var (
	defHMACSecret     string
	defHMACHeaderName = "X-Hub-Signature-256"
	defListenAddr     = ":9002"
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

	hmacSecretValue := flag.String("hmac-secret", defHMACSecret, "your HMAC secret value")
	hmacHeaderName := flag.String(
		"hmac-header-name",
		defHMACHeaderName,
		"name of your signature header",
	)
	listenAddr := flag.String("listen", defListenAddr, "listen addr")
	flag.Parse()

	fmt.Println("hmacSecretValue", *hmacSecretValue)
	fmt.Println("hmacHeaderName", *hmacHeaderName)

	cn := accept.New(
		accept.WithSupportedMediaTypes("text/plain"),
		accept.WithDefaultMediaType("text/plain"),
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/", debugHandlerFunc(cn))

	server := &http.Server{
		Addr:              *listenAddr,
		Handler:           mux,
		ReadTimeout:       defReadTimeout,
		ReadHeaderTimeout: defReadHeaderTimeout,
		WriteTimeout:      defWriteTimeout,
		IdleTimeout:       defIdleTimeout,
	}

	log.Printf("server listening at %s\n", *listenAddr)
	log.Fatal(server.ListenAndServe())
}

func debugHandlerFunc(cn *accept.ContentNegotiation) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		acceptHeader := r.Header.Get("Accept")
		contentType := cn.Negotiate(acceptHeader)

		t := table.NewWriter()
		t.SetOutputMirror(w)
		t.SetTitle("Debug")
		t.AppendRows([]table.Row{
			{"HTTP Method", r.Method},
			{"Matching Content-Type", contentType},
		})
		// t.AppendHeader(table.Row{"Method", "Value"})
		// t.AppendSeparator()
		t.Render()

		t2 := table.NewWriter()
		t2.SetOutputMirror(w)
		t2.SetTitle("Request Headers")
		// t2.AppendHeader(table.Row{"Header", "Value"})

		for _, v := range r.Header {
			t2.AppendRow(table.Row{v})
		}

		t2.Render()

		// w.Header().Set("Content-Type", contentType)
		// log.Printf("acceptHeader: %s\n", acceptHeader)
		// log.Printf("contentType: %s\n", contentType)
	}
}
