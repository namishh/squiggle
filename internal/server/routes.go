package server

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
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
		return c.JSON(http.StatusOK, map[string]string{"message": "Hello, World!"})
	})
	e.POST("/entry", s.handlePost, s.rateLimit, s.checkBanned, s.ttCheck, s.checkOrigin)
	e.GET("/entry", s.listEntries)
	e.GET("/entry/count", s.countEntries)
}
