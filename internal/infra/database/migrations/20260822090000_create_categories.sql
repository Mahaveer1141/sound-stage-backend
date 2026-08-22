-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';

CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE IF NOT EXISTS categories (
	id BIGSERIAL PRIMARY KEY,
	name CITEXT NOT NULL UNIQUE,
	description TEXT,
	created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

	CONSTRAINT categories_name_not_empty CHECK (LENGTH(TRIM(name)) > 0)
);

INSERT INTO categories (name, description) VALUES
    ('Gaming', 'Casual play, esports, squad recruitment, walkthroughs, and live commentary.'),
    ('Music & Audio', 'Jam sessions, listening parties, beat making, podcasting, and open mics.'),
    ('Tech', 'Software development, AI, hardware, troubleshooting, and tech news.'),
    ('Entertainment', 'Movies, TV shows, anime, meme culture, and celebrity news.'),
    ('Learning', 'Study groups, language exchange, history, science, and skill sharing.'),
    ('Business & Career', 'Networking, startup pitches, career advice, finance, and marketing.'),
    ('Sports', 'Live match reactions, fantasy leagues, fitness, esports, and sports news.'),
    ('Lifestyle & Wellness', 'Fitness, mental health, cooking, travel, fashion, and self-improvement.'),
    ('Art & Creativity', 'Digital art, writing, graphic design, content creation, and photography.'),
    ('Chitchat & Hangout', 'General conversations, casual banter, late-night talks, and making friends.'),
    ('News & Debate', 'Current events, politics, philosophy, hot takes, and structured discussions.'),
    ('Other', 'Anything that doesn''t fit into the main categories.');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';

DROP TABLE IF EXISTS categories;
-- +goose StatementEnd
