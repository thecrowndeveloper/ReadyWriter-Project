-- InkHobby / ReadyWriter database schema
-- Run with: psql -d readywriter -f schema.sql

CREATE TABLE IF NOT EXISTS users (
    id        SERIAL PRIMARY KEY,
    name      TEXT NOT NULL,
    email     TEXT NOT NULL UNIQUE,
    password  TEXT NOT NULL,
    role      TEXT NOT NULL DEFAULT 'reader'
              CHECK (role IN ('reader', 'writer'))
);

CREATE TABLE IF NOT EXISTS books (
    id          SERIAL PRIMARY KEY,
    title       TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS chapters (
    id       SERIAL PRIMARY KEY,
    title    TEXT NOT NULL,
    content  TEXT NOT NULL DEFAULT '',
    book_id  INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    status   TEXT NOT NULL DEFAULT 'draft'
             CHECK (status IN ('draft', 'published'))
);

CREATE TABLE IF NOT EXISTS posts (
    id         SERIAL PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content    TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS comments (
    id         SERIAL PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    post_id    INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    content    TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- One like per user per post (also makes toggle/unlike safe)
CREATE TABLE IF NOT EXISTS post_likes (
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    post_id    INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, post_id)
);

CREATE TABLE IF NOT EXISTS saved_books (
    user_id  INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    book_id  INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    saved_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, book_id)
);

CREATE TABLE IF NOT EXISTS reading_history (
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    chapter_id INTEGER NOT NULL REFERENCES chapters(id) ON DELETE CASCADE,
    read_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, chapter_id)
);

CREATE TABLE IF NOT EXISTS earnings (
    id           SERIAL PRIMARY KEY,
    user_id      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    chapter_id   INTEGER NOT NULL REFERENCES chapters(id) ON DELETE CASCADE,
    amount       NUMERIC(10, 2) NOT NULL DEFAULT 0,
    currency     TEXT NOT NULL DEFAULT 'USD',
    earning_type TEXT NOT NULL DEFAULT 'chapter_base',
    status       TEXT NOT NULL DEFAULT 'pending'
                 CHECK (status IN ('pending', 'paid')),
    created_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (chapter_id, earning_type)
);

CREATE TABLE IF NOT EXISTS writing_activity (
    id            SERIAL PRIMARY KEY,
    user_id       INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    chapter_id    INTEGER REFERENCES chapters(id) ON DELETE SET NULL,
    word_count    INTEGER NOT NULL DEFAULT 0,
    activity_type TEXT NOT NULL DEFAULT 'updated',
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_chapters_book   ON chapters (book_id, status);
CREATE INDEX IF NOT EXISTS idx_posts_created   ON posts (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_comments_post   ON comments (post_id);
CREATE INDEX IF NOT EXISTS idx_likes_post      ON post_likes (post_id);
CREATE INDEX IF NOT EXISTS idx_earnings_user   ON earnings (user_id);
CREATE INDEX IF NOT EXISTS idx_activity_user   ON writing_activity (user_id);
