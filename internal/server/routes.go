package server

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/namishh/squiggle/templates"
)

func (s *Server) RegisterRoutes(e *echo.Echo) {
	e.HTTPErrorHandler = s.httpErrorHandler

	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		UnsafeAllowOriginFunc: func(c *echo.Context, origin string) (string, bool, error) {
			if s.cfg.Environment != "prod" {
				return origin, true, nil
			}
			return s.cfg.AllowedOrigin, origin == s.cfg.AllowedOrigin, nil
		},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodOptions},
		AllowHeaders: []string{"Content-Type"},
	}))

	e.GET("/", func(c *echo.Context) error {
		return templates.Home("squiggle").Render(c.Request().Context(), c.Response())
	})

	e.GET("/vault", s.handleVaultPage)

	e.POST("/vault", s.handleVaultLogin, s.vaultLimit)

	admin := e.Group("/admin", s.requireAdmin)
	admin.GET("", s.handleAdminPage)
	admin.GET("/logout", s.handleLogout)
	admin.GET("/all", s.adminListAllEntries)
	admin.GET("/entry", s.adminListEntries)
	admin.POST("/status", s.adminSetStatus)
	admin.POST("/delete", s.adminDeleteEntry)
	admin.GET("/stats", s.adminStats)
	admin.GET("/stream", s.stream)

	e.POST("/entry", s.handlePost, s.rateLimit, s.checkBanned, s.ttCheck, s.checkOrigin)
	e.GET("/entry", s.listEntries)
	e.GET("/entry/count", s.countEntries)

	e.GET("/favicon.ico", func(c *echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})
}
