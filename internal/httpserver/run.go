package httpserver

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/vbyazilim/basichttpdebugger/internal/envutils"
	"github.com/vbyazilim/basichttpdebugger/internal/release"
)

const (
	helpHMACHeaderName              = "name of your signature header, e.g. X-Hub-Signature-256"
	defRawHTTPRequestFileSaveFormat = "%Y-%m-%d-%H%i%s-{hostname}-{url}.raw"
)

// Run creates server instance and runs.
func Run() error {
	listenAddr := flag.String("listen", envutils.GetenvOrDefault("LISTEN", defListenAddr), "listen addr")
	hmacHeaderName := flag.String(
		"hmac-header-name",
		envutils.GetenvOrDefault("HMAC_HEADER_NAME", ""),
		helpHMACHeaderName,
	)
	hmacSecretValue := flag.String("hmac-secret", envutils.GetenvOrDefault("HMAC_SECRET", ""), "your HMAC secret value")
	output := flag.String("output", envutils.GetenvOrDefault("OUTPUT", "stdout"), "output/write responses to")
	color := flag.Bool("color", envutils.GetenvOrDefault("COLOR", false), "enable color")
	saveRawHTTPRequest := flag.Bool(
		"save-raw-http-request",
		envutils.GetenvOrDefault("SAVE_RAW_HTTP_REQUEST", false),
		"enable saving of raw http request",
	)
	saveFormat := flag.String(
		"save-format",
		envutils.GetenvOrDefault("SAVE_FORMAT", defRawHTTPRequestFileSaveFormat),
		"save filename format of raw http",
	)
	version := flag.Bool("version", false, "display version information")
	flag.Parse()

	if *version {
		fmt.Fprintf(flag.CommandLine.Output(), "%s - build: %s\n", release.Version, release.BuildInformation)

		return nil
	}

	server, err := New(
		WithListenAddr(*listenAddr),
		WithHMACHeaderName(*hmacHeaderName),
		WithHMACSecret(*hmacSecretValue),
		WithOutputWriter(*output),
		WithColor(*color),
		WithSaveRawHTTPRequest(*saveRawHTTPRequest),
		WithRawHTTPRequestFileSaveFormat(*saveFormat),
	)
	if err != nil {
		return fmt.Errorf("server init error: %w", err)
	}
	defer func() {
		log.Println("closing output writer")
		if err = server.OutputWriter.Close(); err != nil {
			log.Printf("output close error: %v", err)
		}
	}()

	closed := make(chan struct{})

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig

		log.Println("stopping server")
		if err = server.Stop(); err != nil {
			log.Printf("server stop error: %v", err)
		}

		close(closed)
	}()

	if err = server.Start(); err != nil {
		return fmt.Errorf("server start error: %w", err)
	}

	<-closed
	log.Println("exit, all clear")

	return nil
}
