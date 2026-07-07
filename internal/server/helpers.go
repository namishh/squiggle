package server

import (
	"crypto/sha256"
	"encoding/hex"
	"net"
	"net/url"
	"regexp"
	"strings"

	"github.com/labstack/echo/v5"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
var siteRegex = regexp.MustCompile(`^[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}(/.*)?$`)

func normalizeSite(site string) string {
	site = strings.TrimPrefix(site, "https://")
	site = strings.TrimPrefix(site, "http://")
	return site
}

func hashIP(ip, salt string) string {
	sum := sha256.Sum256([]byte(ip + salt))
	return hex.EncodeToString(sum[:])
}

func validatePostRequest(p *PostRequest) error {
	if p.Name == "" || p.Message == "" {
		return ErrDetails
	}
	if p.Email != "" && !emailRegex.MatchString(p.Email) {
		return ErrEmail
	}
	p.Site = normalizeSite(p.Site)
	if p.Site != "" && !siteRegex.MatchString(p.Site) {
		return ErrSite
	}

	return nil
}

func directIP(c *echo.Context) string {
	ip, _, err := net.SplitHostPort(c.Request().RemoteAddr)
	if err != nil {
		return c.Request().RemoteAddr
	}
	return ip
}

func (s *Server) isOriginAllowed(raw string) bool {
	if raw == "" {
		return false
	}

	allowedUrl, err := url.Parse(s.cfg.AllowedOrigin)
	if err != nil {
		return false
	}

	u, err := url.Parse(raw)
	if err != nil {
		return false
	}

	if u.Scheme != allowedUrl.Scheme {
		return false
	}

	host := u.Hostname()
	allowedHost := allowedUrl.Hostname()

	return host == allowedHost ||
		strings.HasSuffix(host, "."+allowedHost)
}
