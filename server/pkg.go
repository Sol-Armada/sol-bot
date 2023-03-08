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

func New() (*Server, error) {
	r, err := router.New()
	if err != nil {
		return nil, err
	}
	s := &Server{
		ctx:  context.Background(),
		Echo: r,
	}
	return s, nil
}

func (s *Server) Start() error {
	return s.Echo.Start(fmt.Sprintf(":%d", config.GetIntWithDefault("SERVER.PORT", 8080)))
}

func (s *Server) Stop() error {
	return s.Shutdown(s.ctx)
}
