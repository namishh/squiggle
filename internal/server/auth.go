package server

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/namishh/squiggle/templates"
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

	token, err := generateSessionToken()

	if err != nil {
		s.logger.Error("[VAULT] failed to generate session token", "err", err)
		return ErrInternal
	}

	if err := s.redis.Set(c.Request().Context(), "session:"+token, "1", s.cfg.SessionTTL).Err(); err != nil {
		s.logger.Error("[VAULT] failed to store session", "err", err)
		return ErrSessionFailed
	}

	c.SetCookie(&http.Cookie{
		Name:     "admin_session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.cfg.Environment == "prod",
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(s.cfg.SessionTTL.Seconds()),
	})

	return c.NoContent(http.StatusOK)

}

func (s *Server) handleAdminPage(c *echo.Context) error {

	ctx := c.Request().Context()
	return templates.Admin("admin").Render(ctx, c.Response())
}

func (s *Server) handleLogout(c *echo.Context) error {
	cookie, err := c.Cookie("admin_session")
	if err == nil && cookie.Value != "" {
		if err := s.redis.Del(c.Request().Context(), "session:"+cookie.Value).Err(); err != nil {
			s.logger.Error("[LOGOUT] failed to delete session", "err", err)
		}
	}

	c.SetCookie(&http.Cookie{
		Name:     "admin_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   s.cfg.Environment == "prod",
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})

	return c.Redirect(http.StatusSeeOther, "/vault")
}

func (s *Server) handleVaultPage(c *echo.Context) error {
	cookie, err := c.Cookie("admin_session")
	if err == nil && cookie.Value != "" {
		exists, err := s.redis.Exists(c.Request().Context(), "session:"+cookie.Value).Result()
		if err != nil {
			s.logger.Error("[VAULT] session check failed", "err", err)
		} else if exists > 0 {
			return c.Redirect(http.StatusFound, "/admin")
		}
	}

	return templates.Vault().Render(c.Request().Context(), c.Response())
}
