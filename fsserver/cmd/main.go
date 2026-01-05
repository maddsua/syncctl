package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {

	servePort := EnvIntOr("PORT", 80)
	//tlsPort := EnvIntOr("TLS_PORT", 442)

	//	todo: add tls server

	var mux http.ServeMux

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

	//	todo: close fs server instance and wait until it's done
}
