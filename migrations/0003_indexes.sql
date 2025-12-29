-- 0003_indexes.sql
-- Indexes for faster filtering in pop table

CREATE INDEX IF NOT EXISTS idx_pop_host_name
    ON pop (host_name);

CREATE INDEX IF NOT EXISTS idx_pop_poster_type
    ON pop (poster_type);

CREATE INDEX IF NOT EXISTS idx_pop_poster_created_by
    ON pop (poster_created_by);

CREATE INDEX IF NOT EXISTS idx_pop_city
    ON pop (city);

CREATE INDEX IF NOT EXISTS idx_pop_region
    ON pop (region);

CREATE INDEX IF NOT EXISTS idx_pop_type
    ON pop (type);

-- Recommended for sorting + pagination
CREATE INDEX IF NOT EXISTS idx_pop_pop_datetime
    ON pop (pop_datetime DESC);
