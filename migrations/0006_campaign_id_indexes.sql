CREATE INDEX IF NOT EXISTS idx_pop_campaign_id ON pop (campaign_id);
CREATE INDEX IF NOT EXISTS idx_pop_campaign_id_datetime ON pop (campaign_id, pop_datetime);
