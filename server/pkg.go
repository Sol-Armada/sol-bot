package server

import (
	"context"
	"fmt"
	"strconv"

	"github.com/apex/log"
	"github.com/labstack/echo/v4"
	"github.com/sol-armada/sol-bot/config"
	"github.com/sol-armada/sol-bot/router"
)

type Server struct {
	ctx context.Context

	*echo.Echo
}

func New() (*Server, error) {
	r, err := router.New()
	r.HideBanner = true
	r.HidePort = true
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
	port := config.GetIntWithDefault("SERVER.PORT", 8080)
	log.Info("listening on port " + strconv.Itoa(port))
	return s.Echo.Start(fmt.Sprintf(":%d", port))
}

func (s *Server) Stop() error {
	return s.Shutdown(s.ctx)
}
