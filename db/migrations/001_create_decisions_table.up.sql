-- Migration 001: Create decisions table
CREATE TABLE IF NOT EXISTS decisions (
    id BIGSERIAL PRIMARY KEY,
    actor_user_id VARCHAR(255) NOT NULL,
    recipient_user_id VARCHAR(255) NOT NULL,
    liked_recipient BOOLEAN NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(actor_user_id, recipient_user_id)
);

CREATE INDEX IF NOT EXISTS idx_decisions_recipient_liked
    ON decisions(recipient_user_id) 
    WHERE liked_recipient = true;

CREATE INDEX IF NOT EXISTS idx_decisions_actor_recipient
    ON decisions(actor_user_id, recipient_user_id);

CREATE INDEX IF NOT EXISTS idx_decisions_created_at
    ON decisions(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_decisions_recipient_liked_created
    ON decisions(recipient_user_id, liked_recipient, created_at DESC);
