package server

import (
	"fmt"
	"net/http"

	"github.com/apex/log"
	"github.com/sol-armada/admin/config"
	"github.com/sol-armada/admin/router"
)

func Run() error {
	port := fmt.Sprintf(":%d", config.GetIntWithDefault("SERVER.PORT", 8080))
	srv := &http.Server{
		Addr:    port,
		Handler: router.Router(),
	}

	log.WithField("port", port).Info("starting server")

	return srv.ListenAndServe()
}
