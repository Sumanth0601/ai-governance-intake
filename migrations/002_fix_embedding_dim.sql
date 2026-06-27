-- Fix embedding dimension: nvidia/llama-nemotron-embed-vl-1b-v2 returns 2048, not 4096
ALTER TABLE proposals DROP COLUMN IF EXISTS embedding;
ALTER TABLE proposals ADD COLUMN IF NOT EXISTS embedding vector(2048);
