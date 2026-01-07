package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	s4 "github.com/maddsua/syncctl/storage_service"
	"github.com/maddsua/syncctl/storage_service/blobstorage"
	"github.com/maddsua/syncctl/storage_service/handler"
)

func main() {

	servePort := EnvIntOr("PORT", 80)
	//tlsPort := EnvIntOr("TLS_PORT", 442)

	//	todo: add tls server

	storage := blobstorage.Storage{
		RootDir: "data",
	}

	fshandler := handler.NewFsHandler(&storage)

	var mux http.ServeMux

	//	s4 stands for Stipidly-Simple-Storage-Service, btw
	mux.Handle(s4.PrefixV1, http.StripPrefix(strings.TrimRight(s4.PrefixV1, "/"), fshandler))

	srv := http.Server{
		Handler: &mux,
		Addr:    fmt.Sprintf(":%d", servePort),
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
		break
	case err := <-errCh:
		if err != nil {
			slog.Error("SERVER Terminated",
				slog.String("err", err.Error()))
		}
	}

	_ = srv.Close()

	fshandler.Wait()
}
