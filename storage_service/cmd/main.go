package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	s4 "github.com/maddsua/syncctl/storage_service"
	"github.com/maddsua/syncctl/storage_service/blobstorage"
	"github.com/maddsua/syncctl/storage_service/config"
	"github.com/maddsua/syncctl/storage_service/rest_handler"
)

func main() {

	cfgfile := flag.String("config", "/etc/syncctl/server.yml", "Set config file path")
	dataDir := flag.String("datadir", "", "Where to store your stupid files")

	flag.Parse()

	cfg, err := config.ReadConfig(*cfgfile)
	if err != nil {
		slog.Error("Read config",
			slog.String("err", err.Error()))
		os.Exit(1)
	}

	if *dataDir != "" {
		cfg.DataDir = *dataDir
	} else if cfg.DataDir == "" {
		cfg.DataDir = "/var/syncctl/data"
	}

	if cfg.HttpPort < 80 {
		cfg.HttpPort = EnvIntOr("PORT", 80)
	}

	storage := blobstorage.Storage{
		RootDir: cfg.DataDir,
	}

	fshandler := rest_handler.NewHandler(&storage, &cfg.AuthConfig)

	var mux http.ServeMux

	//	s4 stands for Stipidly-Simple-Storage-Service, btw
	mux.Handle(s4.UrlPrefixV1, http.StripPrefix(strings.TrimRight(s4.UrlPrefixV1, "/"), fshandler))

	srv := http.Server{
		Handler: &mux,
		Addr:    fmt.Sprintf(":%d", cfg.HttpPort),
	}

	errCh := make(chan error, 2)

	go func() {

		slog.Info("STARTING http server",
			slog.String("addr", srv.Addr))

		if err := srv.ListenAndServe(); err != nil {
			errCh <- fmt.Errorf("http server: %v", err)
		}
	}()

	exitCh := make(chan os.Signal, 1)
	signal.Notify(exitCh, os.Interrupt, syscall.SIGTERM)

	select {
	case <-exitCh:
		slog.Info("Note: Exiting...")
		_ = srv.Close()
		fshandler.Wait()
	case err := <-errCh:
		if err != nil {
			slog.Error("SERVER Terminated",
				slog.String("err", err.Error()))
			os.Exit(1)
		}
	}
}
