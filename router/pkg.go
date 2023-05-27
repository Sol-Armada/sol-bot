package router

import (
	"io/fs"
	"net/http"

	"github.com/apex/log"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sol-armada/admin/config"
	"github.com/sol-armada/admin/handlers/api"
	"github.com/sol-armada/admin/web"
)

func New() (*echo.Echo, error) {
	e := echo.New()
	e.Debug = false

	if config.GetBool("LOG.DEBUG") {
		e.Debug = true
	}

	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization, "x-user-id"},
		AllowMethods: []string{echo.GET, echo.HEAD, echo.PUT, echo.PATCH, echo.POST, echo.DELETE},
	}))
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(ipWhiteList)

	fsys, err := fs.Sub(web.StaticFiles, "dist")
	if err != nil {
		return nil, err
	}

	indexHandler := http.FileServer(http.FS(fsys))

	e.GET("/", echo.WrapHandler(indexHandler))
	e.GET("/ranks", echo.WrapHandler(http.StripPrefix("/ranks", indexHandler)))
	e.GET("/events", echo.WrapHandler(http.StripPrefix("/events", indexHandler)))
	e.GET("/login", echo.WrapHandler(http.StripPrefix("/login", indexHandler)))
	e.GET("/favicon.ico", echo.WrapHandler(indexHandler))
	e.GET("/logo.png", echo.WrapHandler(indexHandler))
	e.GET("/assets/*", echo.WrapHandler(indexHandler))
	e.GET("/emojis/*", echo.WrapHandler(indexHandler))

	apiGroup := e.Group("/api")

	apiLoginHandler := echo.HandlerFunc(api.Login)

	apiGetUsersHandler := echo.HandlerFunc(api.GetUsers)
	apiGetUserHandler := echo.HandlerFunc(api.GetUser)
	apiUpdateUserHandler := echo.HandlerFunc(api.UpdateUser)
	apiGetRandomUsersHandler := echo.HandlerFunc(api.GetRandomUsers)
	apiIncrementEventHandler := echo.HandlerFunc(api.IncrementEvent)
	apiDecrementEventHandler := echo.HandlerFunc(api.DecrementEvent)

	apiGetEventsHandler := echo.HandlerFunc(api.GetEvents)
	apiCreateEventsHandler := echo.HandlerFunc(api.CreateEvent)
	apiUpdateEventHandler := echo.HandlerFunc(api.UpdateEvent)
	apiDeleteEventHandler := echo.HandlerFunc(api.DeleteEvent)

	apiGetBankBalanceHandler := echo.HandlerFunc(api.GetBankBalance)

	apiGetEmojisHandler := echo.HandlerFunc(api.GetEmojisHandler)

	apiUtilGetXIDHandler := echo.HandlerFunc(api.GenerateXID)

	apiGroup.POST("/login", apiLoginHandler)

	users := apiGroup.Group("/users")
	users.GET("", apiGetUsersHandler)
	users.GET("/:id", apiGetUserHandler)
	users.PUT("/:id", apiUpdateUserHandler)
	users.GET("/random", apiGetRandomUsersHandler)
	users.PUT("/:id/increment", apiIncrementEventHandler)
	users.PUT("/:id/decrement", apiDecrementEventHandler)

	events := apiGroup.Group("/events")
	events.GET("", apiGetEventsHandler)
	events.POST("", apiCreateEventsHandler)
	events.PUT("/:id", apiUpdateEventHandler)
	events.DELETE("/:id", apiDeleteEventHandler)

	bank := apiGroup.Group("/bank")
	bank.GET("/balance", apiGetBankBalanceHandler)

	emojis := apiGroup.Group("/emojis")
	emojis.GET("", apiGetEmojisHandler)

	utils := apiGroup.Group("/utilities")
	utils.GET("", nil)
	utils.GET("/xid", apiUtilGetXIDHandler)

	return e, nil
}

func ipWhiteList(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		ip := c.Request().Header.Get("X-Forwared-For")
		log.Info(ip)
		return next(c)
	}
}
