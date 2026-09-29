-- InkHobby migration: run this on your EXISTING database.
-- Safe to run multiple times (uses IF NOT EXISTS everywhere).
-- Usage:  psql -d readywriter -f migrate.sql

------------------------------------------------------------
-- 1) books: new columns for guest reading
------------------------------------------------------------
ALTER TABLE books
    ADD COLUMN IF NOT EXISTS cover_url    TEXT    NOT NULL DEFAULT '';

ALTER TABLE books
    ADD COLUMN IF NOT EXISTS genre        TEXT    NOT NULL DEFAULT '';

ALTER TABLE books
    ADD COLUMN IF NOT EXISTS status       TEXT    NOT NULL DEFAULT 'ongoing'
                  CHECK (status IN ('ongoing', 'completed'));

ALTER TABLE books
    ADD COLUMN IF NOT EXISTS rating_total INTEGER NOT NULL DEFAULT 0;

ALTER TABLE books
    ADD COLUMN IF NOT EXISTS rating_count INTEGER NOT NULL DEFAULT 0;

-- Backfill description if your old table didn't have it
ALTER TABLE books
    ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '';

------------------------------------------------------------
-- 2) chapters: published/draft status (if missing)
------------------------------------------------------------
ALTER TABLE chapters
    ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'published'
                 CHECK (status IN ('draft', 'published'));

------------------------------------------------------------
-- 3) New tables (created only if they don't exist)
------------------------------------------------------------

-- Reviews: one star rating + written review per user per book
CREATE TABLE IF NOT EXISTS reviews (
    id         SERIAL PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    book_id    INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    rating     INTEGER NOT NULL CHECK (rating BETWEEN 1 AND 5),
    content    TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (user_id, book_id)
);

-- Bookshelf: "Save book" (login required)
CREATE TABLE IF NOT EXISTS saved_books (
    user_id  INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    book_id  INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    saved_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, book_id)
);

-- Reading history (continue reading later)
CREATE TABLE IF NOT EXISTS reading_history (
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    chapter_id INTEGER NOT NULL REFERENCES chapters(id) ON DELETE CASCADE,
    read_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, chapter_id)
);

-- Post likes (makes like/unlike toggle safe)
CREATE TABLE IF NOT EXISTS post_likes (
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    post_id    INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, post_id)
);

------------------------------------------------------------
-- 4) Indexes for the browse/discovery queries
------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_books_genre     ON books (genre);
CREATE INDEX IF NOT EXISTS idx_books_status    ON books (status);
CREATE INDEX IF NOT EXISTS idx_reviews_book    ON reviews (book_id);
CREATE INDEX IF NOT EXISTS idx_saved_user      ON saved_books (user_id);
CREATE INDEX IF NOT EXISTS idx_chapters_book   ON chapters (book_id, status);
CREATE INDEX IF NOT EXISTS idx_likes_post      ON post_likes (post_id);
