package handlers

import (
	"net/http"

	"github.com/sol-armada/admin/web"
)

func IndexHander(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/favicon.ico" {
		rawFile, _ := web.StaticFiles.ReadFile("dist/favicon.ico")
		w.Write(rawFile)
		return
	}
	rawFile, _ := web.StaticFiles.ReadFile("dist/index.html")
	w.Write(rawFile)
}
