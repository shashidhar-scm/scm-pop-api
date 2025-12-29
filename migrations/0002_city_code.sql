-- 0002_city_code.sql
-- Adjust unique constraint

-- 1) Drop old unique constraint (if defined without city)
ALTER TABLE pop
    DROP CONSTRAINT IF EXISTS pop_unique_poster_host_time;

-- 2) Recreate unique constraint including city
ALTER TABLE pop
    ADD CONSTRAINT pop_unique_poster_host_time
        UNIQUE (city, poster_id, host_name, pop_datetime);
