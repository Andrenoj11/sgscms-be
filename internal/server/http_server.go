package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Andrenoj11/sgscms-be/internal/config"
)

type HTTPServer struct {
	server *http.Server

	shutdownTimeout time.Duration
}

func NewHTTPServer(
	cfg *config.Config,
	handler http.Handler,
) *HTTPServer {
	return &HTTPServer{
		server: &http.Server{
			Addr: ":" + cfg.App.Port,

			Handler: handler,

			ReadTimeout:
				cfg.Server.ReadTimeout,

			ReadHeaderTimeout:
				cfg.Server.ReadHeaderTimeout,

			WriteTimeout:
				cfg.Server.WriteTimeout,

			IdleTimeout:
				cfg.Server.IdleTimeout,
		},

		shutdownTimeout:
			cfg.Server.ShutdownTimeout,
	}
}

func (s *HTTPServer) Run(
	ctx context.Context,
) error {
	serverError := make(
		chan error,
		1,
	)

	go func() {
		log.Printf(
			"HTTP server is running on %s",
			s.server.Addr,
		)

		err := s.server.ListenAndServe()

		if err != nil &&
			!errors.Is(
				err,
				http.ErrServerClosed,
			) {
			serverError <- err
			return
		}

		serverError <- nil
	}()

	select {
	case err := <-serverError:
		if err != nil {
			return fmt.Errorf(
				"HTTP server failed: %w",
				err,
			)
		}

		return nil

	case <-ctx.Done():
		log.Println(
			"shutdown signal received",
		)

		shutdownContext, cancel :=
			context.WithTimeout(
				context.Background(),
				s.shutdownTimeout,
			)

		defer cancel()

		if err := s.server.Shutdown(
			shutdownContext,
		); err != nil {
			if closeErr :=
				s.server.Close(); closeErr != nil {
				return fmt.Errorf(
					"force close HTTP server: %w",
					closeErr,
				)
			}

			return fmt.Errorf(
				"graceful shutdown failed: %w",
				err,
			)
		}

		log.Println(
			"HTTP server stopped gracefully",
		)

		return nil
	}
}