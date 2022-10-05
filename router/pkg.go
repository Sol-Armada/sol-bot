package router

import (
	"net/http"
	"path"

	"github.com/gorilla/mux"
	"github.com/sol-armada/admin/handlers"
	"github.com/sol-armada/admin/handlers/api"
	"github.com/sol-armada/admin/users"
	"github.com/sol-armada/admin/web"
)

func Router() http.Handler {
	r := mux.NewRouter()

	apiLoginHandler := http.HandlerFunc(api.Login)
	apiGetUsersHandler := http.HandlerFunc(api.GetUsers)
	apiUserHandler := http.HandlerFunc(api.User)
	apiPutRankHander := http.HandlerFunc(api.SetRank)
	apiCheckLoginHandler := http.HandlerFunc(api.CheckLogin)

	rootPath := r.PathPrefix("/").Subrouter()
	rootPath.HandleFunc("/", handlers.IndexHander)
	rootPath.HandleFunc("/login", handlers.IndexHander)
	rootPath.HandleFunc("/ranks", handlers.IndexHander)
	rootPath.Handle("/assets/{asset}", assets(http.FileServer(http.FS(web.StaticFiles))))
	r.HandleFunc("/health", api.Health)

	apiPath := rootPath.PathPrefix("/api").Subrouter()
	apiPath.Handle("/login", middlewareCORS(apiLoginHandler))

	apiUsersPath := apiPath.PathPrefix("/users").Subrouter()
	apiUsersPath.Handle("/", middlewareCORS(isAdmin(apiGetUsersHandler)))
	apiUsersPath.Handle("/{id}", middlewareCORS(isAdmin(apiUserHandler)))
	apiUsersPath.Handle("/{id}/set-rank", middlewareCORS(isAdmin(apiPutRankHander)))
	apiUsersPath.Handle("/{id}/check-login", middlewareCORS(apiCheckLoginHandler))

	http.Handle("/", r)

	return r
}

func assets(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = path.Join("/dist/", r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func middlewareCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Authorization, X-User-Id")
		// allow options to go through for cors
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userId := r.Header.Get("X-User-Id")
		if userId == "" {
			userId = r.Header.Get("x-user-id")
		}

		if userId == "" {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		admin := users.GetAdmin(userId)
		if admin == nil {
			http.Error(w, "Not Authorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
