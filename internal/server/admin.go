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
	query := s.db.NewSelect().Model(&entries).ColumnExpr("id, name, email, site, message, status, user_agent, custom_data, sentiment_score, hate_score, sexual_score, violence_score, harassment_score, created_at").
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

func (s *Server) adminSetStatus(c *echo.Context) error {
	var req struct {
		Id     string `json:"id"`
		Status string `json:"status"`
	}

	if err := c.Bind(&req); err != nil || req.Id == "" {
		return c.NoContent(http.StatusBadRequest)
	}

	switch req.Status {
	case "visible", "hidden", "spam":
	default:
		return c.NoContent(http.StatusBadRequest)
	}

	if _, err := s.db.NewRaw(`UPDATE entries SET status = ? WHERE id = ?`,
		req.Status, req.Id).Exec(c.Request().Context()); err != nil {
		s.logger.Error("[ADMIN SET STATUS] failed", "err", err, "id", req.Id)
		return ErrInternal
	}
	return c.NoContent(http.StatusOK)
}

func (s *Server) adminStats(c *echo.Context) error {
	ctx := c.Request().Context()
	type Stats struct {
		Total   int `bun:"total" json:"total"`
		Visible int `bun:"visible" json:"visible"`
		Hidden  int `bun:"hidden" json:"hidden"`
		Spam    int `bun:"spam" json:"spam"`
		Today   int `bun:"today" json:"today"`
	}

	var stats Stats
	if err := s.db.NewRaw(`
		SELECT
			count(*) AS total,
			count(*) FILTER (WHERE status = 'visible') AS visible,
			count(*) FILTER (WHERE status = 'hidden') AS hidden,
			count(*) FILTER (WHERE status = 'spam') AS spam,
			count(*) FILTER (WHERE created_at > date_trunc('day', now())) AS today
		FROM entries
	`).Scan(ctx, &stats); err != nil {
		s.logger.Error("[ADMIN STATS] query failed", "err", err)
		return ErrInternal
	}

	var bannedCount int
	if err := s.db.NewRaw(`SELECT count(*) FROM defaulters WHERE banned = true`).Scan(ctx, &bannedCount); err != nil {
		s.logger.Error("[ADMIN STATS] banned query failed", "err", err)
		return ErrInternal
	}

	return c.JSON(http.StatusOK, map[string]any{
		"total": stats.Total, "visible": stats.Visible, "hidden": stats.Hidden,
		"spam": stats.Spam, "today": stats.Today, "banned": bannedCount,
	})
}

func (s *Server) adminDeleteEntry(c *echo.Context) error {
	var req struct {
		Id string `json:"id"`
	}

	if err := c.Bind(&req); err != nil || req.Id == "" {
		return c.NoContent(http.StatusBadRequest)
	}

	res, err := s.db.NewDelete().Table("entries").Where("id = ?", req.Id).Exec(c.Request().Context())

	if err != nil {
		s.logger.Error("[ADMIN DELETE] failed", "err", err, "id", req.Id)
		return ErrInternal
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return c.NoContent(http.StatusNotFound)
	}

	s.logger.Warn("[ADMIN DELETE] entry hard-deleted", "id", req.Id)
	return c.NoContent(http.StatusOK)
}
