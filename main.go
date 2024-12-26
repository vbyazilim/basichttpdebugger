package main

import (
	"log"

	"github.com/vbyazilim/basichttpdebugger/internal/httpserver"
)

func main() {
	if err := httpserver.Run(); err != nil {
		log.Fatal(err)
	}
}
