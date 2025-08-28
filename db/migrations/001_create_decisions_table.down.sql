-- Migration 001 rollback: Drop decisions table
DROP INDEX IF EXISTS idx_decisions_recipient_liked_created;
DROP INDEX IF EXISTS idx_decisions_created_at;
DROP INDEX IF EXISTS idx_decisions_actor_recipient;
DROP INDEX IF EXISTS idx_decisions_recipient_liked;
DROP TABLE IF EXISTS decisions;
