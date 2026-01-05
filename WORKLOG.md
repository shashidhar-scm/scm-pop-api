# pop service WORKLOG

This log explains what the POP service does and why changes were made so future contributors can understand context quickly.

## Service overview
- Go HTTP API that records "Proof of Play" events for kiosks (poster metadata, kiosk location, host, type, etc.) into the `pop` table.
- Exposes REST endpoints under `cmd/server` via handlers in `internal/handlers` to create records, list/filter them, fetch stats, and generate trend data for dashboards (@pop/internal/handlers/pop_handler.go).
- Data access is encapsulated in `internal/repository`, which builds SQL for inserts, paginated queries, stats, and trend aggregations (@pop/internal/repository/pop_repository.go).

## 2025-12-29
- Added click capture support: migrations now add nullable `click_x` / `click_y` columns, models/handlers parse optional coordinates, and repository insert/list/search logic persists them (@pop/migrations/0004_click_position.sql, @pop/internal/models/pop.go, @pop/internal/repository/pop_repository.go, @pop/internal/handlers/pop_handler.go).
- Documented the POP filtering guidance inside tool-gateway instructions (POP totals must use `popList` instead of `popSearch`), so POP endpoints now serve gateway-based analytics workflows.
- Added a configurable, in-memory IP rate limiter. `middleware.RateLimit` now wraps the entire router (before CORS) in `cmd/app/main.go`, using `RATE_LIMIT_WINDOW_SECONDS` and `RATE_LIMIT_MAX` env vars (default 60s/120 requests). K8s ConfigMap injects the defaults so deployments can throttle aggressive agents without code changes (@pop/internal/middleware/middleware.go, @pop/cmd/app/main.go#33-45, @pop/k8s/pop-api.yaml#1-19).

## 2026-01-05
- `/pop/impressions` now returns richer data: repository aggregates both `value` (true impression count) and `play_count` (play time) per poster, and the handler includes a `posters` array exposing `poster_id`, `poster_name`, `impressions`, and `play_time`. Total campaign impressions have also switched to summing `value` instead of `play_count`. (@pop/internal/repository/pop_repository.go, @pop/internal/handlers/pop_handler.go)
