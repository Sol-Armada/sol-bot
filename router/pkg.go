package router

import (
	"io/fs"
	"net/http"

	"github.com/sol-armada/admin/handlers"
	"github.com/sol-armada/admin/handlers/api"
	"github.com/sol-armada/admin/web"
)

func Router() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/login", api.Login)
	mux.HandleFunc("/", handlers.Indexhander)

	assetsFS, _ := fs.Sub(web.StaticFiles, "dist")
	httpFS := http.FileServer(http.FS(assetsFS))
	mux.Handle("/assets/", httpFS)

	return mux
}
