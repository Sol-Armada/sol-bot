package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/apex/log"
	"github.com/sol-armada/admin/config"
	"github.com/sol-armada/admin/router"
)

type Server struct {
	ctx context.Context

	*http.Server
}

func New() *Server {
	port := fmt.Sprintf(":%d", config.GetIntWithDefault("SERVER.PORT", 8080))
	return &Server{
		ctx: context.Background(),
		Server: &http.Server{
			Addr:    port,
			Handler: router.Router(),
		},
	}
}

func (s *Server) Run() error {
	log.WithField("address", s.Addr).Info("starting server")

	return s.ListenAndServe()
}

func (s *Server) Stop() error {
	return s.Shutdown(s.ctx)
}
