package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

func printHeaders(h http.Header) {
	maxLen := 0
	for k := range h {
		if len(k) > maxLen {
			maxLen = len(k)
		}
	}

	fmt.Println("http request headers ...................")
	for k, v := range h {
		dots := strings.Repeat(".", maxLen-len(k)+1)
		fmt.Printf("%s %s %v\n", k, dots, v)
	}
	fmt.Println(strings.Repeat(".", 40))
	fmt.Println()
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println(strings.Repeat("-", 80))
	fmt.Println("method ...... ", r.Method)
	fmt.Println()

	printHeaders(r.Header)

	body, err := io.ReadAll(r.Body)
	defer func() { _ = r.Body.Close() }()

	fmt.Println("error ..................................")
	fmt.Println(err)
	fmt.Println(strings.Repeat(".", 40))
	fmt.Println()

	if err == nil {
		bodyStr := string(body)

		acceptHeader := r.Header.Get("Accept")
		fmt.Printf("acceptHeader: %+v\n", acceptHeader)

		if len(bodyStr) > 0 {
			fmt.Println("body ...................................")

			fmt.Println(bodyStr)

			// var jsonBody map[string]any
			//
			// if err := json.Unmarshal(body, &jsonBody); err != nil {
			// 	fmt.Println(bodyStr)
			// } else {
			// 	prettyJSON, err := json.MarshalIndent(jsonBody, "", "  ")
			//
			// }

			fmt.Println(strings.Repeat(".", 40))
		}

		if *optHMACSecret != "" && *optHMACHeader != "" {
			fmt.Println()
			fmt.Println("hmac validation ........................")

			signature := r.Header.Get(*optHMACHeader)
			h := hmac.New(sha256.New, []byte(*optHMACSecret))
			h.Write(body)

			expectedSignature := "sha256=" + hex.EncodeToString(h.Sum(nil))
			fmt.Println("expected signature...", expectedSignature)
			fmt.Println("incoming signature...", signature)
			fmt.Println("is valid?............", hmac.Equal([]byte(expectedSignature), []byte(signature)))
			fmt.Println(strings.Repeat(".", 40))
		}
	}
	fmt.Println(strings.Repeat("-", 80))
	fmt.Println()
	_, _ = fmt.Fprintf(w, "OK")
}

var (
	optHMACSecret *string
	optHMACHeader *string
	optListenADDR *string
)

type server struct{}

func main() {
	optHMACSecret = flag.String("hmac-secret", "", "HMAC secret")
	optHMACHeader = flag.String("hmac-header-name", "", "Signature response header name")
	optListenADDR = flag.String("listen", ":9002", "Listen address, default: ':9002'")
	flag.Parse()

	srv := &http.Server{
		Addr:         *optListenADDR,
		Handler:      new(server),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	fmt.Println("running server at", *optListenADDR)
	log.Fatal(srv.ListenAndServe())
}
