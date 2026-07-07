package server

import (
	"github.com/go-redis/redis_rate/v10"
	"github.com/labstack/echo/v5"
)

func (s *Server) rateLimit(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		res, err := s.limiter.Allow(c.Request().Context(), directIP(c), redis_rate.PerMinute(5))
		if err != nil {
			return err
		}
		if res.Allowed == 0 {
			return ErrRateLimited
		}
		return next(c)
	}
}

func (s *Server) checkBanned(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		ipHash := hashIP(directIP(c), s.cfg.IPSalt)
		var banned bool

		err := s.db.NewSelect().Table("defaulters").ColumnExpr("banned AND last_offense_at > now() - interval '3 days'").Where("ip_hash = ?", ipHash).Scan(c.Request().Context(), &banned)

		if err != nil {
			s.logger.Warn("[BAN CHECK] error checking", "err", err, "ip_hash", ipHash)
			return next(c) // fail open, don't block legit users on db error
		}

		if banned {
			return ErrBanned
		}

		return next(c)
	}
}

func (s *Server) ttCheck(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		var req EntryRequest
		if err := c.Bind(&req); err != nil {
			return ErrInvalidCaptcha
		}

		token := req.TurnstileToken
		if token == "" {
			token = c.FormValue("cf-turnstile-response")
		}
		if token == "" {
			s.logger.Error("[SECURITY]: Turnstile token missing")
			return ErrInvalidCaptcha
		}

		ok, err := s.ttverify(c.Request().Context(), token, directIP(c))
		if err != nil {
			s.logger.Error("[SECURITY]: Turnstile Verification failed.", "err", err, "ip", directIP(c))
			return ErrInvalidCaptcha
		}
		if !ok {
			s.logger.Error("[SECURITY]: Rate Limit Exceeded.", "ip", directIP(c))
			return ErrRateLimited
		}

		c.Set("postreq", req.PostRequest)
		return next(c)
	}
}

func (s *Server) checkOrigin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		if s.cfg.Environment != "prod" {
			return next(c)
		}

		if s.isOriginAllowed(c.Request().Header.Get("Origin")) ||
			s.isOriginAllowed(c.Request().Header.Get("Referer")) {
			return next(c)
		}

		return ErrInvalidOrigin
	}
}
