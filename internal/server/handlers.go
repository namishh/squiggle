package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

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
		score, flags, err := s.getSentimentScore(bgctx, message, name, site)
		if err != nil {
			s.logger.Error("[SENTIMENT] Error in getting sentiment score", "id", id, "err", err)
			return
		}
		s.logger.Info("[SENTIMENT] scored", "id", id, "score", score, "hate", flags.Hate, "sexual", flags.Sexual, "violence", flags.Violence, "harassment", flags.Harassment)
		s.moderate(id, ipHash, score, flags)

	}(entry.ID, postreq.Message, postreq.Name, postreq.Site, ipHash)

	return c.JSON(http.StatusCreated, map[string]string{"status": "posted"})
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
		CreatedAt     time.Time       `json:"createdAt"`
		TotalCount    int             `bun:"total_count,scanonly" json:"-"`
	}

	var entries []Entry

	includeHidden := c.QueryParam("hidden") == "true"
	search := c.QueryParam("search")
	from := c.QueryParam("from")
	to := c.QueryParam("to")

	page := 1
	if p, err := strconv.Atoi(c.QueryParam("page")); err == nil && p > 0 {
		page = p
	}
	limit := 20
	if l, err := strconv.Atoi(c.QueryParam("limit")); err == nil && l > 0 && l <= 100 {
		limit = l
	}
	offset := (page - 1) * limit

	query := s.db.NewSelect().
		Model(&entries).
		ColumnExpr("id, name, site, message, custom_data, created_at").
		ColumnExpr("count(*) OVER() AS total_count").
		Limit(limit).
		Offset(offset)

	if includeHidden {
		query.Where("(status = ? OR status = ?)", "visible", "hidden")
	} else {
		query.Where("status = ?", "visible")
	}

	if search != "" {
		query.Where(`
			search_vector @@ plainto_tsquery('english', ?)
			OR similarity(message, ?) > 0.3
			OR similarity(name, ?) > 0.3
		`, search, search, search)
		query.OrderExpr(`
			ts_rank(search_vector, plainto_tsquery('english', ?)) DESC,
			GREATEST(similarity(message, ?), similarity(name, ?)) DESC
		`, search, search, search)
	} else {
		query.Order("created_at DESC")
	}

	if from != "" {
		query.Where("created_at >= ?", from)
	}
	if to != "" {
		query.Where("created_at <= ?", to)
	}

	if err := query.Scan(ctx); err != nil {
		s.logger.Error("[LIST ENTRIES] query failed", "err", err)
		return ErrInternal
	}

	totalEntries := 0
	if len(entries) > 0 {
		totalEntries = entries[0].TotalCount
	}
	totalPages := 0
	if totalEntries > 0 {
		totalPages = (totalEntries + limit - 1) / limit
	}

	return c.JSON(http.StatusOK, map[string]any{
		"entries": entries,
		"pagination": map[string]any{
			"page":         page,
			"limit":        limit,
			"totalEntries": totalEntries,
			"totalPages":   totalPages,
			"hasNext":      page < totalPages,
			"hasPrevious":  page > 1,
		},
	})
}

func (s *Server) countEntries(c *echo.Context) error {
	ctx := c.Request().Context()
	includeHidden := c.QueryParam("hidden") == "true"

	var count int
	var err error
	if includeHidden {
		err = s.db.NewRaw(`SELECT COUNT(*) FROM entries WHERE status IN ('visible', 'hidden')`).Scan(ctx, &count)
	} else {
		err = s.db.NewRaw(`SELECT COUNT(*) FROM entries WHERE status = 'visible'`).Scan(ctx, &count)
	}
	if err != nil {
		s.logger.Error("[COUNT ENTRIES] query failed", "err", err)
		return ErrInternal
	}
	return c.JSON(http.StatusOK, map[string]int{"count": count})
}
