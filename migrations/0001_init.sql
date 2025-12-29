-- Initial schema for pop database

CREATE TABLE IF NOT EXISTS pop (
    id BIGSERIAL PRIMARY KEY,
    poster_name VARCHAR(100),
    poster_id VARCHAR(50),
    host_name VARCHAR(50),
    kiosk_name VARCHAR(50),
    poster_type VARCHAR(30),
    pop_datetime TIMESTAMPTZ DEFAULT timezone('UTC', now()),
    poster_created_by INTEGER,
    kiosk_lat DOUBLE PRECISION,
    kiosk_long DOUBLE PRECISION,
    city VARCHAR(20),
    region VARCHAR(20),
    play_count INTEGER,
    value INTEGER,
    type VARCHAR(20),
    url VARCHAR(400),
    created_at TIMESTAMPTZ DEFAULT timezone('UTC', now()),
    updated_at TIMESTAMPTZ DEFAULT timezone('UTC', now()),
    CONSTRAINT pop_unique_poster_host_time
    UNIQUE (poster_id, host_name, pop_datetime)
);
