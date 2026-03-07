-- Migration 016: Add allowed_integrations column to user_agents
-- Stores per-agent integration scope narrowing as JSON array

ALTER TABLE user_agents ADD COLUMN allowed_integrations TEXT NOT NULL DEFAULT '[]';
