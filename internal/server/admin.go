package server

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

func (s *Server) handleVaultLogin(c *echo.Context) error {
	var req struct {
		Password string `json:"password"`
	}
	if err := c.Bind(&req); err != nil || req.Password == "" {
		return c.NoContent(http.StatusBadRequest)
	}

	if req.Password != s.cfg.AdminPassword {
		s.logger.Warn("[VAULT] failed login attempt", "ip", directIP(c))
		return c.NoContent(http.StatusUnauthorized)
	}

	return c.NoContent(http.StatusOK)

}
