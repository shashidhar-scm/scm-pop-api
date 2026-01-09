package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"pop/internal/models"
)

type PopRepository struct {
	db *sql.DB
}

func NewPopRepository(db *sql.DB) *PopRepository {
	return &PopRepository{db: db}
}

type PopFilter struct {
	City            string
	Region          string
	KioskName       string
	HostName        string
	PosterType      string
	PosterName      string
	PosterID        string
	CampaignID      string
	PosterCreatedBy int
	Type            string
	From            time.Time
	To              time.Time
	Page            int
	PageSize        int
}

type PopStatsRequest struct {
	GroupBy string
	City    string
	Region  string
	Metric  string
	Order   string
	Limit   int
	From    time.Time
	To      time.Time
}

type PopStatsRow struct {
	Key        string
	PosterName string
	Metric     int64
	Count      int64
}

type PopTrendRequest struct {
	Dimension string
	Key       string
	Metric    string
	Bucket    string
	From      time.Time
	To        time.Time
}

type PopTrendPoint struct {
	T     time.Time `json:"t"`
	Value int64     `json:"value"`
	Count int64     `json:"count"`
}

type PosterImpression struct {
	PosterID    string
	PosterName  string
	Impressions int64
	PlayTime    int64
}

func (r *PopRepository) PosterImpressionsByCampaign(ctx context.Context, campaignID string) ([]PosterImpression, error) {
	query := `
		SELECT
			poster_id,
			COALESCE(MAX(poster_name), '') AS poster_name,
			COALESCE(SUM(COALESCE(value, 0)), 0) AS impressions,
			COALESCE(SUM(COALESCE(play_count, 0)), 0) AS play_time
		FROM pop
		WHERE campaign_id = $1
		GROUP BY poster_id
		ORDER BY impressions DESC
	`

	rows, err := r.db.QueryContext(ctx, query, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []PosterImpression
	for rows.Next() {
		var pi PosterImpression
		if err := rows.Scan(&pi.PosterID, &pi.PosterName, &pi.Impressions, &pi.PlayTime); err != nil {
			return nil, err
		}
		result = append(result, pi)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *PopRepository) TotalImpressionsByCampaign(ctx context.Context, campaignID string) (int64, error) {
	query := `
		SELECT COALESCE(SUM(COALESCE(value, 0)), 0)
		FROM pop
		WHERE campaign_id = $1
	`

	var total int64
	if err := r.db.QueryRowContext(ctx, query, campaignID).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (r *PopRepository) Create(ctx context.Context, in models.PopInput) (int64, error) {
	query := `
		INSERT INTO pop (
			poster_name,
			poster_id,
			campaign_id,
			host_name,
			kiosk_name,
			poster_type,
			pop_datetime,
			poster_created_by,
			kiosk_lat,
			kiosk_long,
			city,
			region,
			play_count,
			value,
			click_x,
			click_y,
			type,
			url
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
		) RETURNING id
	`

	var id int64
	err := r.db.QueryRowContext(
		ctx,
		query,
		in.PosterName,
		in.PosterID,
		in.CampaignID,
		in.HostName,
		in.KioskName,
		in.PosterType,
		in.PopDatetime,
		in.PosterCreatedBy,
		in.KioskLat,
		in.KioskLong,
		in.City,
		in.Region,
		in.PlayCount,
		in.Value,
		in.ClickX,
		in.ClickY,
		in.Type,
		in.Url,
	).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (r *PopRepository) List(ctx context.Context, f PopFilter) ([]models.PopInput, int64, error) {
	base := `
		SELECT
			poster_name,
			poster_id,
			campaign_id,
			host_name,
			kiosk_name,
			poster_type,
			pop_datetime,
			poster_created_by,
			kiosk_lat,
			kiosk_long,
			city,
			region,
			play_count,
			value,
			click_x,
			click_y,
			type,
			url,
			COUNT(*) OVER() AS total_count
		FROM pop
	`

	var (
		where []string
		args  []any
	)

	add := func(field string, val string) {
		if val == "" {
			return
		}
		args = append(args, val)
		idx := len(args)
		where = append(where, fmt.Sprintf("%s = $%d", field, idx))
	}

	addInt := func(field string, val int) {
		if val == 0 { // assume 0 means "not set"
			return
		}
		args = append(args, val)
		idx := len(args)
		where = append(where, fmt.Sprintf("%s = $%d", field, idx))
	}

	addTime := func(expr string, val time.Time) {
		if val.IsZero() {
			return
		}
		args = append(args, val)
		idx := len(args)
		where = append(where, fmt.Sprintf("%s $%d", expr, idx))
	}


	if !f.From.IsZero() {
		addTime("pop_datetime >=", f.From)
	}
	if !f.To.IsZero() {
		addTime("pop_datetime <", f.To)
	}

	add("city", f.City)
	add("region", f.Region)
	add("kiosk_name", f.KioskName)
	add("host_name", f.HostName)
	add("poster_type", f.PosterType)
	add("poster_name", f.PosterName)
	add("poster_id", f.PosterID)
	add("campaign_id", f.CampaignID)
	addInt("poster_created_by", f.PosterCreatedBy)
	add("type", f.Type)

	query := base
	if len(where) > 0 {
		query = query + " WHERE " + strings.Join(where, " AND ")
	}

	query += " ORDER BY pop_datetime DESC"

	// ---------- Pagination ----------
	pageSize := f.PageSize
	if pageSize <= 0 || pageSize > 1000 {
		pageSize = 100
	}

	page := f.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * pageSize

	limitIdx := len(args) + 1
	offsetIdx := len(args) + 2

	query = fmt.Sprintf("%s LIMIT $%d OFFSET $%d", query, limitIdx, offsetIdx)
	args = append(args, pageSize, offset)
	// -------------------------------
	
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var (
		result     []models.PopInput
		totalCount int64
	)

	for rows.Next() {
		var m models.PopInput
		var total int64
		if err := rows.Scan(
			&m.PosterName,
			&m.PosterID,
			&m.CampaignID,
			&m.HostName,
			&m.KioskName,
			&m.PosterType,
			&m.PopDatetime,
			&m.PosterCreatedBy,
			&m.KioskLat,
			&m.KioskLong,
			&m.City,
			&m.Region,
			&m.PlayCount,
			&m.Value,
			&m.ClickX,
			&m.ClickY,
			&m.Type,
			&m.Url,
			&total,
		); err != nil {
			return nil, 0, err
		}
		totalCount = total
		result = append(result, m)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return result, totalCount, nil
}

func (r *PopRepository) Search(ctx context.Context, q string) ([]models.PopInput, error) {
	pattern := "%" + q + "%"
	query := `
		SELECT
			poster_name,
			poster_id,
			campaign_id,
			host_name,
			kiosk_name,
			poster_type,
			pop_datetime,
			poster_created_by,
			kiosk_lat,
			kiosk_long,
			city,
			region,
			play_count,
			value,
			click_x,
			click_y,
			type,
			url
		FROM pop
		WHERE
			poster_name ILIKE $1 OR
			poster_id ILIKE $1 OR
			campaign_id ILIKE $1 OR
			host_name ILIKE $1 OR
			kiosk_name ILIKE $1 OR
			poster_type ILIKE $1 OR
			poster_created_by::text ILIKE $1 OR
			city ILIKE $1 OR
			region ILIKE $1 OR
			type ILIKE $1 OR
			url ILIKE $1
	`

	query += " ORDER BY pop_datetime DESC"

	rows, err := r.db.QueryContext(ctx, query, pattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.PopInput
	for rows.Next() {
		var m models.PopInput
		if err := rows.Scan(
			&m.PosterName,
			&m.PosterID,
			&m.CampaignID,
			&m.HostName,
			&m.KioskName,
			&m.PosterType,
			&m.PopDatetime,
			&m.PosterCreatedBy,
			&m.KioskLat,
			&m.KioskLong,
			&m.City,
			&m.Region,
			&m.PlayCount,
			&m.Value,
			&m.ClickX,
			&m.ClickY,
			&m.Type,
			&m.Url,
		); err != nil {
			return nil, err
		}
		result = append(result, m)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *PopRepository) Stats(ctx context.Context, req PopStatsRequest) ([]PopStatsRow, error) {
	groupExpr := ""
	selectName := ""
	switch req.GroupBy {
	case "poster":
		groupExpr = "poster_id"
		selectName = "MAX(poster_name) AS poster_name"
	case "device":
		groupExpr = "host_name"
		selectName = "'' AS poster_name"
	case "kiosk":
		groupExpr = "kiosk_name"
		selectName = "'' AS poster_name"
	default:
		return nil, fmt.Errorf("invalid group_by")
	}

	metricExpr := ""
	switch req.Metric {
	case "plays":
		metricExpr = "SUM(COALESCE(play_count, 0))"
	case "value", "clicks":
		metricExpr = "SUM(COALESCE(value, 0))"
	case "count":
		metricExpr = "COUNT(*)"
	default:
		return nil, fmt.Errorf("invalid metric")
	}

	orderDir := "DESC"
	switch req.Order {
	case "top":
		orderDir = "DESC"
	case "bottom":
		orderDir = "ASC"
	default:
		return nil, fmt.Errorf("invalid order")
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 1000 {
		limit = 1000
	}

	args := []any{req.From, req.To}
	where := "pop_datetime >= $1 AND pop_datetime < $2"
	if strings.TrimSpace(req.City) != "" {
		args = append(args, req.City)
		where = where + fmt.Sprintf(" AND LOWER(city) = LOWER($%d)", len(args))
	}
	if strings.TrimSpace(req.Region) != "" {
		args = append(args, req.Region)
		where = where + fmt.Sprintf(" AND LOWER(region) = LOWER($%d)", len(args))
	}
	args = append(args, limit)
	limitIdx := len(args)

	query := fmt.Sprintf(`
		SELECT
			%s::text AS key,
			%s,
			%s AS metric,
			COUNT(*) AS count
		FROM pop
		WHERE %s
		GROUP BY %s
		ORDER BY metric %s
		LIMIT $%d
	`, groupExpr, selectName, metricExpr, where, groupExpr, orderDir, limitIdx)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []PopStatsRow
	for rows.Next() {
		var rrow PopStatsRow
		if err := rows.Scan(&rrow.Key, &rrow.PosterName, &rrow.Metric, &rrow.Count); err != nil {
			return nil, err
		}
		out = append(out, rrow)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

func (r *PopRepository) Trend(ctx context.Context, req PopTrendRequest) ([]PopTrendPoint, error) {
	if strings.TrimSpace(req.Key) == "" {
		return nil, fmt.Errorf("key is required")
	}

	dimCol := ""
	switch req.Dimension {
	case "poster":
		dimCol = "poster_id"
	case "device":
		dimCol = "host_name"
	case "city":
		dimCol = "city"
	default:
		return nil, fmt.Errorf("invalid dimension")
	}

	bucket := "day"
	switch req.Bucket {
	case "":
		bucket = "day"
	case "hour", "day", "week", "month":
		bucket = req.Bucket
	default:
		return nil, fmt.Errorf("invalid bucket")
	}

	metricExpr := ""
	switch req.Metric {
	case "plays":
		metricExpr = "SUM(COALESCE(play_count, 0))"
	case "clicks", "value":
		metricExpr = "SUM(COALESCE(value, 0))"
	case "count":
		metricExpr = "COUNT(*)"
	default:
		return nil, fmt.Errorf("invalid metric")
	}

	query := fmt.Sprintf(`
		SELECT
			date_trunc('%s', pop_datetime) AS t,
			%s AS value,
			COUNT(*) AS count
		FROM pop
		WHERE pop_datetime >= $1 AND pop_datetime < $2 AND %s = $3
		GROUP BY 1
		ORDER BY 1
	`, bucket, metricExpr, dimCol)

	rows, err := r.db.QueryContext(ctx, query, req.From, req.To, req.Key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []PopTrendPoint
	for rows.Next() {
		var p PopTrendPoint
		if err := rows.Scan(&p.T, &p.Value, &p.Count); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}
