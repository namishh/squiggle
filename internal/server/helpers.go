package server

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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

func generateSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}

const dominantFlagSQL = `
CASE
  WHEN hate_score >= sexual_score AND hate_score >= violence_score AND hate_score >= harassment_score AND hate_score > 0 THEN 'hate'
  WHEN sexual_score >= hate_score AND sexual_score >= violence_score AND sexual_score >= harassment_score AND sexual_score > 0 THEN 'sexual'
  WHEN violence_score >= hate_score AND violence_score >= sexual_score AND violence_score >= harassment_score AND violence_score > 0 THEN 'violence'
  WHEN harassment_score >= hate_score AND harassment_score >= sexual_score AND harassment_score >= violence_score AND harassment_score > 0 THEN 'harassment'
  ELSE 'none'
END`

func cacheKey(parts ...any) string {
	quoted := make([]string, len(parts))
	for i, p := range parts {
		quoted[i] = fmt.Sprintf("%q", p)
	}
	sum := sha256.Sum256([]byte(strings.Join(quoted, "\x00")))
	return hex.EncodeToString(sum[:])
}

func sanitizePromptField(s string) string {
	s = strings.ReplaceAll(s, "<", "")
	s = strings.ReplaceAll(s, ">", "")
	return s
}

type promptTags struct {
	name, website, comment string
}

func newPromptTags() (promptTags, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return promptTags{}, err
	}
	suffix := hex.EncodeToString(b)
	return promptTags{
		name:    "name-" + suffix,
		website: "website-" + suffix,
		comment: "comment-" + suffix,
	}, nil
}

func (t promptTags) wrap(tag, value string) string {
	return "<" + tag + ">" + value + "</" + tag + ">"
}

var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
var numericRegex = regexp.MustCompile(`^\d+$`)

func isUUID(s string) bool    { return uuidRegex.MatchString(s) }
func isNumeric(s string) bool { return numericRegex.MatchString(s) }
