-- Enhanced database schema for Arch Search System
-- Migration: 001_initial_schema.sql

-- Search analytics table
CREATE TABLE IF NOT EXISTS search_queries (
    id SERIAL PRIMARY KEY,
    query_text TEXT NOT NULL,
    user_session VARCHAR(255),
    results_count INTEGER DEFAULT 0,
    clicked_result_id VARCHAR(255),
    search_timestamp TIMESTAMP DEFAULT NOW(),
    response_time_ms INTEGER,
    user_agent TEXT,
    ip_address INET,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- User feedback table
CREATE TABLE IF NOT EXISTS user_feedback (
    id SERIAL PRIMARY KEY,
    query_id INTEGER REFERENCES search_queries(id) ON DELETE CASCADE,
    feedback_type VARCHAR(50) NOT NULL CHECK (feedback_type IN ('helpful', 'not_helpful', 'partially_helpful')),
    feedback_text TEXT,
    user_session VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Content metadata cache table
CREATE TABLE IF NOT EXISTS content_metadata (
    id SERIAL PRIMARY KEY,
    wiki_page_title VARCHAR(255) UNIQUE NOT NULL,
    alchemyst_context_id VARCHAR(255),
    error_patterns TEXT[],
    content_hash VARCHAR(64),
    page_url TEXT,
    content_type VARCHAR(50) DEFAULT 'wiki_page',
    last_crawled TIMESTAMP,
    last_updated TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW(),
    is_active BOOLEAN DEFAULT TRUE,
    crawl_status VARCHAR(20) DEFAULT 'pending' CHECK (crawl_status IN ('pending', 'crawling', 'completed', 'failed')),
    word_count INTEGER,
    section_count INTEGER
);

-- Search performance analytics
CREATE TABLE IF NOT EXISTS search_analytics (
    id SERIAL PRIMARY KEY,
    date_hour TIMESTAMP NOT NULL,
    total_searches INTEGER DEFAULT 0,
    avg_response_time_ms INTEGER DEFAULT 0,
    successful_searches INTEGER DEFAULT 0,
    failed_searches INTEGER DEFAULT 0,
    unique_sessions INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(date_hour)
);

-- Popular search terms
CREATE TABLE IF NOT EXISTS popular_queries (
    id SERIAL PRIMARY KEY,
    query_text TEXT NOT NULL,
    search_count INTEGER DEFAULT 1,
    avg_results_count DECIMAL(5,2) DEFAULT 0,
    avg_response_time_ms INTEGER DEFAULT 0,
    last_searched TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(query_text)
);

-- Wiki page sections for better search granularity
CREATE TABLE IF NOT EXISTS wiki_sections (
    id SERIAL PRIMARY KEY,
    content_metadata_id INTEGER REFERENCES content_metadata(id) ON DELETE CASCADE,
    section_title VARCHAR(255) NOT NULL,
    section_content TEXT NOT NULL,
    section_order INTEGER NOT NULL,
    alchemyst_context_id VARCHAR(255),
    error_patterns TEXT[],
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- System health monitoring
CREATE TABLE IF NOT EXISTS system_health (
    id SERIAL PRIMARY KEY,
    service_name VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('healthy', 'degraded', 'unhealthy')),
    response_time_ms INTEGER,
    error_message TEXT,
    checked_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_search_queries_timestamp ON search_queries(search_timestamp);
CREATE INDEX IF NOT EXISTS idx_search_queries_session ON search_queries(user_session);
CREATE INDEX IF NOT EXISTS idx_search_queries_text_gin ON search_queries USING gin(to_tsvector('english', query_text));

CREATE INDEX IF NOT EXISTS idx_content_metadata_title ON content_metadata(wiki_page_title);
CREATE INDEX IF NOT EXISTS idx_content_metadata_updated ON content_metadata(last_updated);
CREATE INDEX IF NOT EXISTS idx_content_metadata_active ON content_metadata(is_active);
CREATE INDEX IF NOT EXISTS idx_content_metadata_status ON content_metadata(crawl_status);

CREATE INDEX IF NOT EXISTS idx_user_feedback_query ON user_feedback(query_id);
CREATE INDEX IF NOT EXISTS idx_user_feedback_type ON user_feedback(feedback_type);
CREATE INDEX IF NOT EXISTS idx_user_feedback_session ON user_feedback(user_session);

CREATE INDEX IF NOT EXISTS idx_wiki_sections_metadata ON wiki_sections(content_metadata_id);
CREATE INDEX IF NOT EXISTS idx_wiki_sections_order ON wiki_sections(content_metadata_id, section_order);

CREATE INDEX IF NOT EXISTS idx_search_analytics_date ON search_analytics(date_hour);
CREATE INDEX IF NOT EXISTS idx_popular_queries_count ON popular_queries(search_count DESC);
CREATE INDEX IF NOT EXISTS idx_popular_queries_last_searched ON popular_queries(last_searched);

CREATE INDEX IF NOT EXISTS idx_system_health_service ON system_health(service_name);
CREATE INDEX IF NOT EXISTS idx_system_health_checked ON system_health(checked_at);

-- Create functions for auto-updating timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for auto-updating timestamps
CREATE TRIGGER update_search_queries_updated_at BEFORE UPDATE ON search_queries FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_user_feedback_updated_at BEFORE UPDATE ON user_feedback FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_content_metadata_updated_at BEFORE UPDATE ON content_metadata FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_popular_queries_updated_at BEFORE UPDATE ON popular_queries FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_wiki_sections_updated_at BEFORE UPDATE ON wiki_sections FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Insert initial system health data
INSERT INTO system_health (service_name, status, response_time_ms) VALUES 
('postgresql', 'healthy', 0),
('redis', 'healthy', 0),
('nats', 'healthy', 0),
('alchemyst', 'healthy', 0)
ON CONFLICT DO NOTHING;

-- Insert initial content metadata for key Arch Wiki pages
INSERT INTO content_metadata (wiki_page_title, page_url, error_patterns, content_type) VALUES 
('General_troubleshooting', 'https://wiki.archlinux.org/title/General_troubleshooting', ARRAY['error', 'failed', 'cannot', 'problem', 'issue'], 'wiki_page'),
('Pacman/Troubleshooting', 'https://wiki.archlinux.org/title/Pacman/Troubleshooting', ARRAY['pacman', 'package', 'dependency', 'conflict', 'keyring'], 'wiki_page'),
('NetworkManager', 'https://wiki.archlinux.org/title/NetworkManager', ARRAY['network', 'connection', 'wifi', 'ethernet', 'dns'], 'wiki_page'),
('Systemd', 'https://wiki.archlinux.org/title/Systemd', ARRAY['systemd', 'service', 'unit', 'failed', 'start'], 'wiki_page'),
('GRUB', 'https://wiki.archlinux.org/title/GRUB', ARRAY['grub', 'boot', 'bootloader', 'rescue'], 'wiki_page')
ON CONFLICT (wiki_page_title) DO NOTHING;