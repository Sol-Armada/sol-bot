package server

import (
	"context"
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/sol-armada/admin/config"
	"github.com/sol-armada/admin/router"
)

type Server struct {
	ctx context.Context

	*echo.Echo
}

func New() *Server {
	r := router.New()

	return &Server{
		ctx:  context.Background(),
		Echo: r,
	}
}

func (s *Server) Start() error {
	return s.Echo.Start(fmt.Sprintf(":%d", config.GetIntWithDefault("SERVER.PORT", 8080)))
}

func (s *Server) Stop() error {
	return s.Shutdown(s.ctx)
}
