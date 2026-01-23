package httpserver

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/vbyazilim/basichttpdebugger/internal/envutils"
	"github.com/vbyazilim/basichttpdebugger/internal/release"
	"github.com/vbyazilim/basichttpdebugger/internal/requeststore"
	"github.com/vbyazilim/basichttpdebugger/internal/webui"
)

const (
	helpHMACHeaderName              = "name of your signature header, e.g. X-Hub-Signature-256"
	helpSecretTokenHeaderName       = "name of your secret token header, e.g. X-Gitlab-Token"
	defRawHTTPRequestFileSaveFormat = "%Y-%m-%d-%H%i%s-{hostname}-{url}.raw"
	defWebDashboardMaxRequests      = 50
)

// Run creates server instance and runs.
func Run() error {
	listenAddr := flag.String("listen", envutils.GetenvOrDefault("LISTEN", defListenAddr), "listen addr")

	hmacSecretValue := flag.String("hmac-secret", envutils.GetenvOrDefault("HMAC_SECRET", ""), "your HMAC secret value")
	hmacHeaderName := flag.String(
		"hmac-header-name",
		envutils.GetenvOrDefault("HMAC_HEADER_NAME", ""),
		helpHMACHeaderName,
	)

	secretToken := flag.String("secret-token", envutils.GetenvOrDefault("SECRET_TOKEN", ""), "your secret token value")
	secretTokenHeaderName := flag.String(
		"secret-token-header-name",
		envutils.GetenvOrDefault("SECRET_TOKEN_HEADER_NAME", ""),
		helpSecretTokenHeaderName,
	)

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
	webListen := flag.String(
		"web-listen",
		envutils.GetenvOrDefault("WEB_LISTEN", ""),
		"web dashboard listen addr (default: debug port + 1)",
	)
	version := flag.Bool("version", false, "display version information")
	flag.Parse() //nolint:revive

	if *version {
		fmt.Fprintf(flag.CommandLine.Output(), "%s - build: %s\n", release.Version, release.BuildInformation)

		return nil
	}

	store := requeststore.New(defWebDashboardMaxRequests)

	server, err := New(
		WithListenAddr(*listenAddr),
		WithHMACHeaderName(*hmacHeaderName),
		WithHMACSecret(*hmacSecretValue),
		WithSecretToken(*secretToken),
		WithSecretTokenHeaderName(*secretTokenHeaderName),
		WithOutputWriter(*output),
		WithColor(*color),
		WithSaveRawHTTPRequest(*saveRawHTTPRequest),
		WithRawHTTPRequestFileSaveFormat(*saveFormat),
		WithStore(store),
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

	webListenAddr := *webListen
	if webListenAddr == "" {
		webListenAddr = calculateWebPort(*listenAddr)
	}
	webServer := webui.New(store, webListenAddr)

	go func() {
		if webErr := webServer.Start(); webErr != nil {
			log.Printf("web dashboard error: %v", webErr)
		}
	}()

	closed := make(chan struct{})

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig

		log.Println("stopping servers")

		if webErr := webServer.Stop(); webErr != nil {
			log.Printf("web dashboard stop error: %v", webErr)
		}

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

func calculateWebPort(listenAddr string) string {
	parts := strings.Split(listenAddr, ":")
	if len(parts) != 2 {
		return ":9003"
	}

	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return ":9003"
	}

	return fmt.Sprintf("%s:%d", parts[0], port+1)
}
