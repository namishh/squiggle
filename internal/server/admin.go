package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/uptrace/bun"
)

type AdminEntry struct {
	bun.BaseModel   `bun:"table:entries"`
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	DominantFlag    string          `json:"dominantFlag" bun:"dominant_flag,scanonly"`
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

	status := c.QueryParam("status")
	profanity := c.QueryParam("profanity")
	sortOrder := c.QueryParam("sort")
	search := strings.TrimSpace(c.QueryParam("search"))
	from := c.QueryParam("from")
	to := c.QueryParam("to")

	page := 1
	if p, err := strconv.Atoi(c.QueryParam("page")); err == nil && p > 0 {
		page = p
	}
	limit := 50
	if l, err := strconv.Atoi(c.QueryParam("limit")); err == nil && l > 0 && l <= 200 {
		limit = l
	}
	offset := (page - 1) * limit

	version, _ := s.redis.Get(ctx, "admin:entries:version").Result()
	key := "admin:entries:cache:" + cacheKey(status, profanity, sortOrder, search, from, to, page, limit, version)

	if cached, err := s.redis.Get(ctx, key).Result(); err == nil {
		return c.JSONBlob(http.StatusOK, []byte(cached))
	}

	var entries []AdminEntry
	query := s.db.NewSelect().Model(&entries).
		ColumnExpr("id, name, email, site, message, status, user_agent, custom_data, sentiment_score, hate_score, sexual_score, violence_score, harassment_score, created_at").
		ColumnExpr(dominantFlagSQL + " AS dominant_flag").
		ColumnExpr("count(*) OVER() AS total_count").
		Limit(limit).Offset(offset)

	if status != "" {
		query.Where("status = ?", status)
	}
	if profanity != "" {
		query.Where(dominantFlagSQL+" = ?", profanity)
	}

	if search != "" {
		switch {
		case isUUID(search):
			query.Where("id = ?", search)
		case isNumeric(search):
			days, _ := strconv.Atoi(search)
			query.Where("created_at >= now() - (? || ' days')::interval", days)
		default:
			site := normalizeSite(search)
			query.Where(`
					search_vector @@ plainto_tsquery('english', ?)
					OR similarity(message, ?) > 0.25
					OR similarity(name, ?) > 0.25
					OR similarity(site, ?) > 0.25
					OR site ILIKE ?
				`, search, search, search, site, "%"+site+"%")
		}
	}

	if from != "" {
		query.Where("created_at >= ?", from)
	}
	if to != "" {
		query.Where("created_at <= ?", to)
	}

	if sortOrder == "oldest" {
		query.Order("created_at asc")
	} else {
		query.Order("created_at desc")
	}

	if err := query.Scan(ctx); err != nil {
		s.logger.Error("[ADMIN ENTRIES] query failed", "err", err)
		return ErrInternal
	}

	total := 0
	if len(entries) > 0 {
		total = entries[0].TotalCount
	}
	totalPages := 0
	if total > 0 {
		totalPages = (total + limit - 1) / limit
	}

	resp := map[string]any{
		"entries": entries,
		"pagination": map[string]any{
			"page":         page,
			"limit":        limit,
			"totalEntries": total,
			"totalPages":   totalPages,
		},
	}

	if body, err := json.Marshal(resp); err == nil {
		s.redis.Set(ctx, key, body, 30*time.Second)
	}

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) bumpEntriesCache(ctx context.Context) {
	if err := s.redis.Incr(ctx, "admin:entries:version").Err(); err != nil {
		s.logger.Warn("[ADMIN CACHE] version bump failed", "err", err)
	}
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
	s.bumpEntriesCache(c.Request().Context())
	s.publish(c.Request().Context(), "status", map[string]any{"id": req.Id, "status": req.Status})
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
	s.bumpEntriesCache(c.Request().Context())
	s.publish(c.Request().Context(), "delete", map[string]any{"id": req.Id})
	return c.NoContent(http.StatusOK)
}
