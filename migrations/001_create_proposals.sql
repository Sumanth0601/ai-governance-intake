CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS proposals (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title           TEXT NOT NULL,
    description     TEXT NOT NULL,
    data_sources    TEXT NOT NULL,
    intended_use    TEXT NOT NULL,
    embedding       vector(2048),
    risk_score      TEXT NOT NULL DEFAULT 'pending', -- 'low' | 'medium' | 'high' | 'critical' | 'duplicate'
    scorecard       JSONB,
    duplicate_of    UUID REFERENCES proposals(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
