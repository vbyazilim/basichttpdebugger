package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
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

type server struct{}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println(strings.Repeat("-", 80))
	fmt.Println("method ...... ", r.Method)
	fmt.Println()

	printHeaders(r.Header)

	body, err := io.ReadAll(r.Body)

	fmt.Println("error ..................................")
	fmt.Println(err)
	fmt.Println(strings.Repeat(".", 40))
	fmt.Println()

	if err == nil {
		bodyStr := string(body)
		if len(bodyStr) > 0 {
			fmt.Println("body ...................................")
			fmt.Println(bodyStr)
			fmt.Println(strings.Repeat(".", 40))
		}
	}
	fmt.Println(strings.Repeat("-", 80))
	fmt.Println()
	fmt.Fprintf(w, "OK")
}

func main() {
	host := os.Getenv("HOST")
	if host == "" {
		host = ":9002"
	}

	srv := &http.Server{
		Addr:         host,
		Handler:      new(server),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	fmt.Println("running server at", host)
	log.Fatal(srv.ListenAndServe())
}
