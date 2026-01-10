package main

import (
	"flag"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	s4 "github.com/maddsua/syncctl/storage_service"
	"github.com/maddsua/syncctl/storage_service/blobstorage"
	"github.com/maddsua/syncctl/storage_service/config"
	"github.com/maddsua/syncctl/storage_service/rest_handler"
	"github.com/maddsua/syncctl/utils"
)

func main() {

	cfgfile := flag.String("config", "/etc/syncctl/server.yml", "Set config file path")
	dataDir := flag.String("data", "", "Where to store your stupid files")

	flag.Parse()

	cfg, err := config.ReadConfig(*cfgfile)
	if err != nil {
		slog.Error("Read config",
			slog.String("err", err.Error()))
		os.Exit(1)
	}

	storage := blobstorage.Storage{
		RootDir: selectString(*dataDir, os.Getenv("S4_DATA_DIR"), cfg.DataDir, "/var/syncctl/data"),
	}

	fshandler := rest_handler.NewHandler(&storage, &cfg.AuthConfig)

	var mux http.ServeMux

	//	s4 stands for Stipidly-Simple-Storage-Service, btw
	mux.Handle(s4.UrlPrefixV1, http.StripPrefix(strings.TrimRight(s4.UrlPrefixV1, "/"), fshandler))

	plainSrv := http.Server{
		Handler: &mux,
		Addr:    fmt.Sprintf(":%d", selectPortNumber(utils.EnvInt("S4_PORT"), cfg.HttpPort, 44_080)),
	}

	tlsSrv := http.Server{
		Handler:   &mux,
		Addr:      fmt.Sprintf(":%d", selectPortNumber(utils.EnvInt("S4_TLS_PORT"), cfg.TlsPort, 44_443)),
		TLSConfig: setupSelfSignedTlsOrDie(),
	}

	errCh := make(chan error, 2)

	go func() {

		slog.Info("Note: Starting http server",
			slog.String("addr", plainSrv.Addr))

		if err := plainSrv.ListenAndServe(); err != nil {
			errCh <- fmt.Errorf("http server: %v", err)
		}
	}()

	go func() {

		slog.Info("Note: Starting tls server",
			slog.String("addr", tlsSrv.Addr))

		if err := tlsSrv.ListenAndServeTLS("", ""); err != nil {
			errCh <- fmt.Errorf("tls server: %v", err)
		}
	}()

	exitCh := make(chan os.Signal, 1)
	signal.Notify(exitCh, os.Interrupt, syscall.SIGTERM)

	select {
	case <-exitCh:
		slog.Info("Note: Exiting...")
		_ = plainSrv.Close()
		_ = tlsSrv.Close()
		fshandler.Wait()
	case err := <-errCh:
		slog.Error("Terminated",
			slog.String("reason", err.Error()))
		os.Exit(1)
	}
}

func selectPortNumber(opts ...int) int {
	return utils.SelectValue(func(val int) bool {
		return val > 0 && val < math.MaxUint16
	}, opts...)
}

func selectString(opts ...string) string {
	return utils.SelectValue(func(val string) bool {
		return val != ""
	}, opts...)
}
