package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"pop/internal/models"
	"pop/internal/repository"
)

type PopHandler struct {
	repo *repository.PopRepository
}

type PopListResponse struct {
	Items    []models.PopInput `json:"items"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
}

type PopStatsResponse struct {
	GroupBy  string                 `json:"group_by"`
	Metric   string                 `json:"metric"`
	Order    string                 `json:"order"`
	Limit    int                    `json:"limit"`
	From     time.Time              `json:"from"`
	To       time.Time              `json:"to"`
	Items    []repository.PopStatsRow `json:"items"`
}

type PopTrendResponse struct {
	Dimension string                  `json:"dimension"`
	Key       string                  `json:"key"`
	Metric    string                  `json:"metric"`
	Bucket    string                  `json:"bucket"`
	From      time.Time               `json:"from"`
	To        time.Time               `json:"to"`
	Points    []repository.PopTrendPoint `json:"points"`
}

type PosterImpressionResponse struct {
	PosterID    string `json:"poster_id"`
	PosterName  string `json:"poster_name"`
	Impressions int64  `json:"impressions"`
	PlayTime    int64  `json:"play_time"`
}

type PopImpressionsResponse struct {
	CampaignID  string                       `json:"campaign_id"`
	Impressions int64                        `json:"impressions"`
	Posters     []PosterImpressionResponse   `json:"posters"`
}


func NewPopHandler(repo *repository.PopRepository) *PopHandler {
	return &PopHandler{repo: repo}
}

func (h *PopHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var input models.PopInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		fmt.Println("Error decoding JSON body:", err)
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	// fmt.Println("Inside Create ", input)
	// Basic validation for required fields
	if input.HostName == "" {
		writeError(w, http.StatusBadRequest, "host_name is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	id, err := h.repo.Create(ctx, input)
	if err != nil {
		fmt.Println("Error creating pop:", err)
		// crude check for unique violation without pulling in pg-specific types
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "duplicate poster for host at this time")
			return
		}

		writeError(w, http.StatusInternalServerError, "failed to insert pop")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":      id,
		"message": "created",
	})
}

func qInt(q url.Values, key string) int {
    s := q.Get(key)
    if s == "" {
        return 0
    }
    i, _ := strconv.Atoi(s)
    return i
}

func qIntDefault(q url.Values, key string, def int) int {
	v := qInt(q, key)
	if v == 0 {
		return def
	}
	return v
}


func (h *PopHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	pageSize, _ := strconv.Atoi(q.Get("page_size"))

	f := repository.PopFilter{
		City:            q.Get("city"),
		Region:          q.Get("region"),
		KioskName:       q.Get("kiosk_name"),
		HostName:        q.Get("host_name"),
		PosterType:      q.Get("poster_type"),
		PosterName:      q.Get("poster_name"),
		PosterID:        q.Get("poster_id"),
		CampaignID:      q.Get("campaign_id"),
		PosterCreatedBy: qInt(q, "poster_created_by"),
		Type:            q.Get("type"),
		Page:            page,
		PageSize:        pageSize,
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	items, total, err := h.repo.List(ctx, f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch pop records")
		return
	}

	resp := PopListResponse{
		Items:    items,
		Total:    total,
		Page:     f.Page,
		PageSize: f.PageSize,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *PopHandler) Impressions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	campaignID := strings.TrimSpace(r.URL.Query().Get("campaign_id"))
	if campaignID == "" {
		writeError(w, http.StatusBadRequest, "campaign_id query parameter is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	total, err := h.repo.TotalImpressionsByCampaign(ctx, campaignID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch impressions")
		return
	}

	posterRows, err := h.repo.PosterImpressionsByCampaign(ctx, campaignID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch poster level impressions")
		return
	}

	posters := make([]PosterImpressionResponse, 0, len(posterRows))
	for _, p := range posterRows {
		posters = append(posters, PosterImpressionResponse{
			PosterID:    p.PosterID,
			PosterName:  p.PosterName,
			Impressions: p.Impressions,
			PlayTime:    p.PlayTime,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(PopImpressionsResponse{
		CampaignID:  campaignID,
		Impressions: total,
		Posters:     posters,
	})
}

func (h *PopHandler) Trend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	dimension := strings.TrimSpace(q.Get("dimension"))
	if dimension == "" {
		dimension = "device"
	}

	key := strings.TrimSpace(q.Get("key"))
	if key == "" {
		writeError(w, http.StatusBadRequest, "key query parameter is required")
		return
	}

	metric := strings.TrimSpace(q.Get("metric"))
	if metric == "" {
		metric = "plays"
	}

	bucket := strings.TrimSpace(q.Get("bucket"))
	if bucket == "" {
		bucket = "day"
	}

	lastDays := qIntDefault(q, "last_days", 30)
	if lastDays <= 0 {
		lastDays = 30
	}

	to := time.Now().UTC()
	from := to.AddDate(0, 0, -lastDays)

	if s := strings.TrimSpace(q.Get("from")); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid from; expected RFC3339")
			return
		}
		from = t.UTC()
	}
	if s := strings.TrimSpace(q.Get("to")); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid to; expected RFC3339")
			return
		}
		to = t.UTC()
	}
	if !from.Before(to) {
		writeError(w, http.StatusBadRequest, "from must be before to")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	points, err := h.repo.Trend(ctx, repository.PopTrendRequest{
		Dimension: dimension,
		Key:       key,
		Metric:    metric,
		Bucket:    bucket,
		From:      from,
		To:        to,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp := PopTrendResponse{
		Dimension: dimension,
		Key:       key,
		Metric:    metric,
		Bucket:    bucket,
		From:      from,
		To:        to,
		Points:    points,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *PopHandler) Search(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeError(w, http.StatusBadRequest, "q query parameter is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	items, err := h.repo.Search(ctx, q)
	if err != nil {
		fmt.Println("Error searching pop:", err)
		writeError(w, http.StatusInternalServerError, "failed to search pop records")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(items)
}

func (h *PopHandler) Stats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	groupBy := strings.TrimSpace(q.Get("group_by"))
	if groupBy == "" {
		groupBy = "poster"
	}

	city := strings.TrimSpace(q.Get("city"))
	region := strings.TrimSpace(q.Get("region"))

	metric := strings.TrimSpace(q.Get("metric"))
	if metric == "" {
		metric = "plays"
	}

	order := strings.TrimSpace(q.Get("order"))
	if order == "" {
		order = "top"
	}

	limit := qIntDefault(q, "limit", 10)
	lastDays := qIntDefault(q, "last_days", 30)
	if lastDays <= 0 {
		lastDays = 30
	}

	to := time.Now().UTC()
	from := to.AddDate(0, 0, -lastDays)

	if s := strings.TrimSpace(q.Get("from")); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid from; expected RFC3339")
			return
		}
		from = t.UTC()
	}
	if s := strings.TrimSpace(q.Get("to")); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid to; expected RFC3339")
			return
		}
		to = t.UTC()
	}
	if !from.Before(to) {
		writeError(w, http.StatusBadRequest, "from must be before to")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	items, err := h.repo.Stats(ctx, repository.PopStatsRequest{
		GroupBy: groupBy,
		City:    city,
		Region:  region,
		Metric:  metric,
		Order:   order,
		Limit:   limit,
		From:    from,
		To:      to,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp := PopStatsResponse{
		GroupBy: groupBy,
		Metric:  metric,
		Order:   order,
		Limit:   limit,
		From:    from,
		To:      to,
		Items:   items,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	var msg string
	if e := errors.Unwrap(err); e != nil {
		msg = e.Error()
	} else {
		msg = err.Error()
	}
	return strings.Contains(msg, "duplicate key") || strings.Contains(msg, "23505")
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": msg,
	})
}
