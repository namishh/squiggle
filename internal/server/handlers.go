package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/uptrace/bun"
)

type PostRequest struct {
	Name       string          `json:"name"`
	Email      string          `json:"email"`
	Message    string          `json:"message"`
	Site       string          `json:"site"`
	CustomData json.RawMessage `json:"customData,omitempty"`
}

type EntryRequest struct {
	PostRequest
	TurnstileToken string `json:"turnstileToken"`
}

func (s *Server) handlePost(c *echo.Context) error {
	postreq, ok := c.Get("postreq").(PostRequest)
	if !ok {
		return ErrInternal
	}

	ip := directIP(c)
	ipHash := hashIP(ip, s.cfg.IPSalt)
	userAgent := c.Request().UserAgent()

	if err := validatePostRequest(&postreq); err != nil {
		return err
	}

	type Entry struct {
		bun.BaseModel `bun:"table:entries"`

		ID         string          `bun:"id,pk,default:gen_random_uuid()"`
		Name       string          `bun:"name"`
		Email      string          `bun:"email"`
		CustomData json.RawMessage `bun:"custom_data,type:jsonb"`
		Site       string          `bun:"site"`
		Message    string          `bun:"message"`
		IPHash     string          `bun:"ip_hash"`
		UserAgent  string          `bun:"user_agent"`
	}

	entry := Entry{
		Name:       postreq.Name,
		Email:      postreq.Email,
		Site:       postreq.Site,
		Message:    postreq.Message,
		CustomData: postreq.CustomData,
		IPHash:     ipHash,
		UserAgent:  userAgent,
	}

	_, err := s.db.NewInsert().Model(&entry).Exec(c.Request().Context())
	if err != nil {
		return ErrEntryPost
	}
	s.logger.Info("[INSERT] New guestbook entry", "id", entry.ID, "name", postreq.Name, "ip_hash", ipHash)

	go func(id, message, name, site, ipHash string) {
		bgctx := context.Background()
		score, err := s.getSentimentScore(bgctx, message, name, site)
		if err != nil {
			s.logger.Error("[SENTIMENT] Error in getting sentiment score", "id", id, "err", err)
			return
		}
		s.moderate(id, ipHash, score)

	}(entry.ID, postreq.Message, postreq.Name, postreq.Site, ipHash)

	return c.JSON(http.StatusCreated, map[string]string{"status": "posted"})
}

func (s *Server) countEntries(c *echo.Context) error {
	ctx := c.Request().Context()
	var count int

	err := s.db.NewRaw(`SELECT COUNT(*) FROM entries WHERE status = 'visible'`).Scan(ctx, &count)
	if err != nil {
		return ErrInternal
	}

	return c.JSON(http.StatusOK, map[string]int{"count": count})
}

func (s *Server) listEntries(c *echo.Context) error {
	ctx := c.Request().Context()

	type Entry struct {
		bun.BaseModel `bun:"table:entries"`
		ID            string          `json:"id"`
		Name          string          `json:"name"`
		Site          string          `json:"site"`
		Message       string          `json:"message"`
		CustomData    json.RawMessage `json:"customData,omitempty"`
	}

	var visible []Entry
	var hidden []Entry

	if err := s.db.NewSelect().Model(&visible).
		Where("status = ?", "visible").
		Order("created_at DESC").
		Scan(ctx); err != nil {
		return ErrInternal
	}

	if err := s.db.NewSelect().Model(&hidden).
		Where("status = ?", "hidden").
		Order("created_at DESC").
		Scan(ctx); err != nil {
		return ErrInternal
	}

	return c.JSON(http.StatusOK, map[string][]Entry{
		"visible": visible,
		"hidden":  hidden,
	})
}
