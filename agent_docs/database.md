# Database & Schema

## Database Schema

Core tables: `users`, `bingo_cards`, `bingo_items`, `friendships`, `reactions`, `suggestions`, `sessions`

Email verification tables: `email_verification_tokens`, `magic_link_tokens`, `password_reset_tokens`

**Users table key columns:**
- `username` - Unique (case-insensitive) user display name
- `searchable` - Boolean, opt-in flag for appearing in friend search (default: false)

Migrations in `migrations/` directory using numeric prefix ordering.

## Tech Stack
- **Database**: PostgreSQL with pgx/v5 driver
- **Cache/Sessions**: Redis with go-redis/v9
- **Migrations**: golang-migrate/migrate/v4
