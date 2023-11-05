package router

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/apex/log"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sol-armada/admin/config"
	"github.com/sol-armada/admin/handlers/api"
	"github.com/sol-armada/admin/web"
)

func New() (*echo.Echo, error) {
	logger := log.WithField("func", "router")
	e := echo.New()

	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:    true,
		LogStatus: true,
		LogMethod: true,
		LogValuesFunc: func(c echo.Context, values middleware.RequestLoggerValues) error {
			logger.WithFields(log.Fields{
				"uri":    values.URI,
				"status": values.Status,
				"method": values.Method,
			}).Debug("request")

			return nil
		},
	}))
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization, "X-Token-Auth"},
		AllowMethods: []string{echo.GET, echo.HEAD, echo.PUT, echo.PATCH, echo.POST, echo.DELETE},
	}))
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(echojwt.WithConfig(echojwt.Config{
		SigningKey: []byte(config.GetString("SERVER.SECRET")),
		ContextKey: "token",
		Skipper: func(c echo.Context) bool {
			return c.Request().URL.Path == "/api/health" || c.Request().URL.Path == "/api/login" || !strings.Contains(c.Request().URL.Path, "api")
		},
		ErrorHandler: func(c echo.Context, err error) error {
			if err := c.JSON(http.StatusUnauthorized, "unautorized"); err != nil {
				return err
			}
			return err
		},
	}))

	fsys, err := fs.Sub(web.StaticFiles, "dist")
	if err != nil {
		return nil, err
	}

	indexHandler := http.FileServer(http.FS(fsys))

	e.GET("/", echo.WrapHandler(indexHandler))
	e.GET("/dashboard", echo.WrapHandler(http.StripPrefix("/dashboard", indexHandler)))
	e.GET("/ranks", echo.WrapHandler(http.StripPrefix("/ranks", indexHandler)))
	e.GET("/events", echo.WrapHandler(http.StripPrefix("/events", indexHandler)))
	e.GET("/events/edit", echo.WrapHandler(http.StripPrefix("/events/edit", indexHandler)))
	e.GET("/events/new", echo.WrapHandler(http.StripPrefix("/events/new", indexHandler)))
	e.GET("/login", echo.WrapHandler(http.StripPrefix("/login", indexHandler)))
	e.GET("/favicon.ico", echo.WrapHandler(indexHandler))
	e.GET("/logo.png", echo.WrapHandler(indexHandler))
	e.GET("/logo-ts3.png", echo.WrapHandler(indexHandler))
	e.GET("/assets/*", echo.WrapHandler(indexHandler))
	e.GET("/emojis/*", echo.WrapHandler(indexHandler))

	apiGroup := e.Group("/api")

	apiHealthHandler := echo.HandlerFunc(health)

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

	apiGetEventTemplatesHandler := echo.HandlerFunc(api.GetEventTemplates)
	apiCreateEventTemplatesHandler := echo.HandlerFunc(api.CreateEventTemplate)
	apiUpdateEventTemplatesHandler := echo.HandlerFunc(api.UpdateEventTemplate)
	apiDeleteEventTemplatesHandler := echo.HandlerFunc(api.DeleteEventTemplate)

	apiGetBankBalanceHandler := echo.HandlerFunc(api.GetBankBalance)

	apiGetEmojisHandler := echo.HandlerFunc(api.GetEmojisHandler)

	apiUtilGetXIDHandler := echo.HandlerFunc(api.GenerateXID)

	apiGroup.GET("/health", apiHealthHandler)

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

	templates := apiGroup.Group("/event_templates")
	templates.GET("", apiGetEventTemplatesHandler)
	templates.POST("", apiCreateEventTemplatesHandler)
	templates.PUT("/:id", apiUpdateEventTemplatesHandler)
	templates.DELETE("/:id", apiDeleteEventTemplatesHandler)

	bank := apiGroup.Group("/bank")
	bank.GET("/balance", apiGetBankBalanceHandler)

	emojis := apiGroup.Group("/emojis")
	emojis.GET("", apiGetEmojisHandler)

	utils := apiGroup.Group("/utilities")
	utils.GET("", nil)
	utils.GET("/xid", apiUtilGetXIDHandler)

	return e, nil
}
