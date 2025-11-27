-- Drop triggers first
DROP TRIGGER IF EXISTS update_bingo_cards_updated_at ON bingo_cards;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables in reverse order of creation (respecting foreign keys)
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS suggestions;
DROP TABLE IF EXISTS reactions;
DROP TABLE IF EXISTS friendships;
DROP TABLE IF EXISTS bingo_items;
DROP TABLE IF EXISTS bingo_cards;
DROP TABLE IF EXISTS users;
