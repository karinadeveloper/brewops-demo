-- Demo-only: tracks per-session and (by aggregation) global daily usage of
-- POST /inventory/suggest so the public demo can't run up an unbounded
-- OpenAI bill. Does not exist in the real BrewOps product — see CLAUDE.md's
-- "DEMO MODE" section. Only written to when DEMO_MODE=true.
CREATE TABLE ai_usage (
  session_id UUID NOT NULL,
  usage_date DATE NOT NULL,
  request_count INT NOT NULL DEFAULT 0,
  PRIMARY KEY (session_id, usage_date)
);
