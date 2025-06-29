-- Initialize database with basic tables
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Search analytics
CREATE TABLE IF NOT EXISTS search_queries (
    id SERIAL PRIMARY KEY,
    query_text TEXT NOT NULL,
    user_session VARCHAR(255),
    results_count INTEGER,
    clicked_result_id VARCHAR(255),
    search_timestamp TIMESTAMP DEFAULT NOW(),
    response_time_ms INTEGER,
    created_at TIMESTAMP DEFAULT NOW()
);

-- User feedback
CREATE TABLE IF NOT EXISTS user_feedback (
    id SERIAL PRIMARY KEY,
    query_id INTEGER REFERENCES search_queries(id),
    feedback_type VARCHAR(50), -- 'helpful', 'not_helpful', 'partially_helpful'
    feedback_text TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Content metadata cache
CREATE TABLE IF NOT EXISTS content_metadata (
    id SERIAL PRIMARY KEY,
    wiki_page_title VARCHAR(255) UNIQUE,
    alchemyst_context_id VARCHAR(255),
    error_patterns TEXT[],
    content_hash VARCHAR(64),
    last_updated TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_search_queries_timestamp ON search_queries(search_timestamp);
CREATE INDEX IF NOT EXISTS idx_search_queries_session ON search_queries(user_session);
CREATE INDEX IF NOT EXISTS idx_content_metadata_title ON content_metadata(wiki_page_title);
CREATE INDEX IF NOT EXISTS idx_content_metadata_updated ON content_metadata(last_updated);
CREATE INDEX IF NOT EXISTS idx_user_feedback_query ON user_feedback(query_id);

-- Insert initial data
INSERT INTO content_metadata (wiki_page_title, content_hash, error_patterns) VALUES 
('General_troubleshooting', '', ARRAY['error', 'failed', 'cannot']),
('Pacman/Troubleshooting', '', ARRAY['pacman', 'package', 'dependency']),
('NetworkManager', '', ARRAY['network', 'connection', 'wifi'])
ON CONFLICT (wiki_page_title) DO NOTHING;