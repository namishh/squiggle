package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/uptrace/bun"
)

type AdminEntry struct {
	bun.BaseModel   `bun:"table:entries"`
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Email           string          `json:"email"`
	Site            string          `json:"site"`
	Message         string          `json:"message"`
	Status          string          `json:"status"`
	IPHash          string          `json:"ipHash" bun:"ip_hash"`
	UserAgent       string          `json:"userAgent" bun:"user_agent"`
	CustomData      json.RawMessage `json:"customData,omitempty" bun:"custom_data"`
	SentimentScore  float64         `json:"sentimentScore" bun:"sentiment_score"`
	HateScore       int             `json:"hateScore" bun:"hate_score"`
	SexualScore     int             `json:"sexualScore" bun:"sexual_score"`
	ViolenceScore   int             `json:"violenceScore" bun:"violence_score"`
	HarassmentScore int             `json:"harassmentScore" bun:"harassment_score"`
	CreatedAt       time.Time       `json:"createdAt" bun:"created_at"`
	TotalCount      int             `bun:"total_count,scanonly" json:"-"`
}

func (s *Server) adminListEntries(c *echo.Context) error {
	ctx := c.Request().Context()

	status := c.QueryParam("status") // "", "visible", "hidden", "spam"
	page := 1
	if p, err := strconv.Atoi(c.QueryParam("page")); err == nil && p > 0 {
		page = p
	}
	limit := 50
	if l, err := strconv.Atoi(c.QueryParam("limit")); err == nil && l > 0 && l <= 200 {
		limit = l
	}
	offset := (page - 1) * limit

	var entries []AdminEntry
	query := s.db.NewSelect().Model(&entries).ColumnExpr("id, name, email, site, message, status, ip_hash, user_agent, custom_data, sentiment_score, hate_score, sexual_score, violence_score, harassment_score, created_at").
		ColumnExpr("count(*) OVER() AS total_count").Order("created_at desc").Limit(limit).Offset(offset)

	if status != "" {
		query.Where("status = ?", status)
	}

	if err := query.Scan(ctx); err != nil {
		s.logger.Error("[ADMIN ENTRIES] query failed", "err", err)
		return ErrInternal
	}

	total := 0
	if len(entries) > 0 {
		total = entries[0].TotalCount
	}

	return c.JSON(http.StatusOK, map[string]any{
		"entries": entries,
		"pagination": map[string]any{
			"page":         page,
			"limit":        limit,
			"totalEntries": total,
		},
	})
}

func (s *Server) adminListAllEntries(c *echo.Context) error {
	ctx := c.Request().Context()

	var entries []AdminEntry

	if err := s.db.NewSelect().
		Model(&entries).
		Order("created_at DESC").
		Scan(ctx); err != nil {
		s.logger.Error("[ADMIN ENTRIES] query failed", "err", err)
		return ErrInternal
	}

	return c.JSON(http.StatusOK, map[string]any{
		"entries": entries,
	})
}
