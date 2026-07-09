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
	e.GET("/admin", s.handleAdminPage, s.requireAdmin)
	e.GET("/admin/logout", s.handleLogout, s.requireAdmin)

	e.GET("/admin/all", s.adminListAllEntries, s.requireAdmin)
	e.GET("/admin/entry", s.adminListEntries, s.requireAdmin)
	e.POST("/admin/status", s.adminSetStatus, s.requireAdmin)
	e.GET("/admin/stats", s.adminStats, s.requireAdmin)

	e.POST("/entry", s.handlePost, s.rateLimit, s.checkBanned, s.ttCheck, s.checkOrigin)
	e.GET("/entry", s.listEntries)
	e.GET("/entry/count", s.countEntries)

	e.GET("/favicon.ico", func(c *echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})
}
