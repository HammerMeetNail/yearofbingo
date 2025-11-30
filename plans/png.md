# PNG Share Links Plan

## Overview

Allow users to generate shareable PNG images of their bingo cards with public links. PNGs are generated server-side and served directly, making them embeddable in Discord, Twitter, emails, and anywhere that supports image URLs.

## User Stories

1. As a user, I want to share a PNG of my bingo card on social media
2. As a user, I want to control whether my progress (completions) is visible in the shared image
3. As a user, I want to set when my share link expires
4. As a user, I want to revoke share links I no longer want public
5. As a user, I want the shared image to look good and include my username and stats

## Design Decisions

- **Generation**: Server-side using Go image library (`fogleman/gg`)
- **Storage**: Local filesystem in `/data/shares/` (or object storage for scale)
- **Link format**: `yearofbingo.com/share/{token}.png` - direct image URL
- **Expiration**: User-configurable, default 18 months
- **Content**: Full card with grid, username, year, completion stats, bingo count
- **Completions**: User chooses whether to show or hide progress

## Database Schema

### New Table: `card_shares`

```sql
CREATE TABLE card_shares (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    card_id UUID NOT NULL REFERENCES bingo_cards(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    share_token VARCHAR(32) NOT NULL UNIQUE,  -- URL-safe random token
    show_completions BOOLEAN NOT NULL DEFAULT true,
    expires_at TIMESTAMP WITH TIME ZONE,      -- NULL means never expires
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_accessed_at TIMESTAMP WITH TIME ZONE,
    access_count INTEGER DEFAULT 0
);

CREATE INDEX idx_card_shares_token ON card_shares(share_token);
CREATE INDEX idx_card_shares_card_id ON card_shares(card_id);
CREATE INDEX idx_card_shares_user_id ON card_shares(user_id);
```

### Migration

```
migrations/000X_card_shares.up.sql
migrations/000X_card_shares.down.sql
```

## API Endpoints

### Share Management (requires auth)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/cards/{id}/shares` | List all share links for a card |
| POST | `/api/cards/{id}/shares` | Create a new share link |
| DELETE | `/api/cards/{id}/shares/{shareId}` | Revoke a specific share link |

### Public Access (no auth)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/share/{token}.png` | Serve the PNG image directly |
| GET | `/share/{token}` | HTML page with OG tags for rich previews |

### Create Share Request

```json
POST /api/cards/{cardId}/shares
{
    "show_completions": true,
    "expires_in_days": 548  // ~18 months, or null for never
}
```

### Create Share Response

```json
{
    "share": {
        "id": "uuid",
        "card_id": "uuid",
        "share_token": "EXAMPLE_SHARE_TOKEN",
        "show_completions": true,
        "expires_at": "2027-05-30T00:00:00Z",
        "created_at": "2025-11-30T00:00:00Z",
        "url": "https://yearofbingo.com/share/abc123xyz789.png",
        "preview_url": "https://yearofbingo.com/share/abc123xyz789"
    }
}
```

### List Shares Response

```json
{
    "shares": [
        {
            "id": "uuid",
            "share_token": "EXAMPLE_SHARE_TOKEN",
            "show_completions": true,
            "expires_at": "2027-05-30T00:00:00Z",
            "created_at": "2025-11-30T00:00:00Z",
            "last_accessed_at": "2025-12-01T12:00:00Z",
            "access_count": 42,
            "url": "https://yearofbingo.com/share/abc123xyz789.png"
        }
    ]
}
```

## PNG Generation

### Image Layout

```
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│                   HammerMeetNail - 2025                     │
│                                                             │
├─────┬─────┬─────┬─────┬─────┬─────────────────────────────┤
│  B  │  I  │  N  │  G  │  O  │                             │
├─────┼─────┼─────┼─────┼─────┤                             │
│     │     │     │     │     │                             │
│ ✓   │     │ ✓   │     │     │     12/24 Complete          │
│     │     │     │     │     │     2 Bingos                │
├─────┼─────┼─────┼─────┼─────┤                             │
│     │     │     │     │     │                             │
│     │ ✓   │     │     │ ✓   │                             │
│     │     │     │     │     │                             │
├─────┼─────┼─────┼─────┼─────┤                             │
│     │     │     │     │     │                             │
│     │     │FREE │     │     │                             │
│     │     │     │     │     │                             │
├─────┼─────┼─────┼─────┼─────┤                             │
│     │     │     │     │     │                             │
│ ✓   │     │     │ ✓   │     │                             │
│     │     │     │     │     │                             │
├─────┼─────┼─────┼─────┼─────┤                             │
│     │     │     │     │     │     yearofbingo.com         │
│     │ ✓   │     │     │ ✓   │                             │
│     │     │     │     │     │                             │
└─────┴─────┴─────┴─────┴─────┴─────────────────────────────┘
```

### Image Specifications

- **Dimensions**: 1200x630px (optimal for social media OG images)
- **Format**: PNG with transparency support
- **Colors**: Match site color scheme (CSS variables → Go constants)
- **Font**: Use a readable sans-serif (Inter, Roboto, or system font)
- **Cell text**: Truncate long text with ellipsis, max 3-4 lines per cell

### Visual Elements

1. **Header**: Username and year, centered
2. **Grid**: 5x5 with B-I-N-G-O column headers
3. **Cells**:
   - Item text (truncated if needed)
   - Completion checkmark overlay (if show_completions=true and completed)
   - FREE space in center
4. **Stats panel** (right side):
   - X/24 Complete (if show_completions=true)
   - X Bingos (if show_completions=true)
   - Or "Progress hidden" if show_completions=false
5. **Footer**: yearofbingo.com branding

### Go Image Library

Use `github.com/fogleman/gg` for 2D graphics:

```go
import "github.com/fogleman/gg"

func GenerateCardPNG(card *models.BingoCard, showCompletions bool) ([]byte, error) {
    const width = 1200
    const height = 630

    dc := gg.NewContext(width, height)

    // Background
    dc.SetHexColor("#1a1a2e")
    dc.Clear()

    // Draw header, grid, cells, stats...

    var buf bytes.Buffer
    dc.EncodePNG(&buf)
    return buf.Bytes(), nil
}
```

## Implementation

### Phase 1: Database & Models

1. Create migration for `card_shares` table
2. Add `CardShare` model in `internal/models/card_share.go`
3. Add `ShareService` in `internal/services/share.go`
   - `Create(cardID, userID, showCompletions, expiresInDays) (*CardShare, error)`
   - `List(cardID, userID) ([]CardShare, error)`
   - `GetByToken(token) (*CardShare, error)`
   - `Delete(shareID, userID) error`
   - `IncrementAccessCount(shareID) error`
   - `CleanupExpired() (int, error)` - for scheduled cleanup

### Phase 2: PNG Generation

1. Add `github.com/fogleman/gg` dependency
2. Create `internal/services/image.go`:
   - `GenerateCardPNG(card *models.BingoCard, username string, showCompletions bool) ([]byte, error)`
3. Design color scheme constants matching CSS variables
4. Implement text truncation with ellipsis
5. Add completion checkmark overlay
6. Add stats panel rendering
7. Add header and footer branding

### Phase 3: Share API Endpoints

1. Add handlers in `internal/handlers/share.go`:
   - `HandleListShares`
   - `HandleCreateShare`
   - `HandleDeleteShare`
2. Register routes in `cmd/server/main.go`
3. Token generation: 22 chars URL-safe base64 (16 bytes entropy)

### Phase 4: Public Routes

1. Add public handlers (no auth required):
   - `HandleSharePNG` - serves PNG with appropriate headers
   - `HandleSharePreview` - HTML page with OG meta tags
2. PNG route: `/share/{token}.png`
   - Content-Type: image/png
   - Cache-Control: public, max-age=3600 (1 hour)
   - Check expiration, return 404 if expired
   - Increment access count
3. Preview route: `/share/{token}`
   - HTML page with OG tags for rich social previews
   - Links to the PNG
   - "View on Year of Bingo" button (if card is visible to friends)

### Phase 5: Frontend UI

1. Add "Share" button to finalized card view:
   - Opens share modal
2. Share modal:
   - Toggle: "Show my progress" (default: on)
   - Dropdown: Expiration (1 month, 6 months, 18 months, 1 year, never)
   - "Create Share Link" button
3. After creation:
   - Show share URL with copy button
   - Show PNG preview thumbnail
   - "Open in new tab" link
4. Manage shares section (on card view or profile):
   - List existing shares with stats
   - Delete/revoke button for each

### Phase 6: Caching & Cleanup

1. **PNG Caching**:
   - Generate PNG on first access, cache to disk
   - Cache key: `{token}_{card_updated_at}.png`
   - Invalidate when card changes (item completed, etc.)
   - Or regenerate on each access (simpler, slightly slower)
2. **Expired share cleanup**:
   - Daily cron job or startup task
   - Delete expired `card_shares` rows
   - Delete orphaned PNG files

## Open Graph Meta Tags

For rich previews on social media, the `/share/{token}` HTML page includes:

```html
<meta property="og:title" content="HammerMeetNail's 2025 Bingo Card">
<meta property="og:description" content="12/24 complete, 2 bingos! Check out my Year of Bingo card.">
<meta property="og:image" content="https://yearofbingo.com/share/abc123xyz789.png">
<meta property="og:image:width" content="1200">
<meta property="og:image:height" content="630">
<meta property="og:url" content="https://yearofbingo.com/share/abc123xyz789">
<meta property="og:type" content="website">
<meta name="twitter:card" content="summary_large_image">
```

## Security Considerations

1. **Token entropy**: 16 bytes = 128 bits, unguessable
2. **Rate limiting**: Consider limits on PNG generation to prevent abuse
3. **Expiration enforcement**: Always check expiration before serving
4. **No auth bypass**: Share links only expose the PNG, not card editing
5. **Privacy**: Users control what's shared (completions visible or not)
6. **Card visibility**: Share links work even if card is private to friends (user explicitly shared it)

## UI Mockups

### Share Button on Card View

```
┌─────────────────────────────────────────────────────────────┐
│ My 2025 Bingo Card                    [Share] [Edit] [...]  │
└─────────────────────────────────────────────────────────────┘
```

### Create Share Modal

```
┌─────────────────────────────────────────┐
│ Share Your Bingo Card                 ✕ │
├─────────────────────────────────────────┤
│                                         │
│ ┌─────────────────────────────────────┐ │
│ │ [✓] Show my progress                │ │
│ │     (completions and bingo count)   │ │
│ └─────────────────────────────────────┘ │
│                                         │
│ Link expires                            │
│ ┌─────────────────────────────────────┐ │
│ │ 18 months                         ▼ │ │
│ └─────────────────────────────────────┘ │
│                                         │
│           [Cancel] [Create Share Link]  │
└─────────────────────────────────────────┘
```

### Share Created Modal

```
┌─────────────────────────────────────────┐
│ Share Link Created!                   ✕ │
├─────────────────────────────────────────┤
│                                         │
│ ┌─────────────────────────────────────┐ │
│ │ [PNG Preview Thumbnail]             │ │
│ └─────────────────────────────────────┘ │
│                                         │
│ Direct image link:                      │
│ ┌─────────────────────────────────────┐ │
│ │ yearofbingo.com/share/abc123.png 📋│ │
│ └─────────────────────────────────────┘ │
│                                         │
│ Preview page (with social embed):       │
│ ┌─────────────────────────────────────┐ │
│ │ yearofbingo.com/share/abc123     📋│ │
│ └─────────────────────────────────────┘ │
│                                         │
│ Expires: May 30, 2027                   │
│                                         │
│          [Open in New Tab] [Done]       │
└─────────────────────────────────────────┘
```

### Manage Shares (on Card View)

```
┌─────────────────────────────────────────────────────────────┐
│ Shared Links                                                │
├─────────────────────────────────────────────────────────────┤
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ abc123... • Shows progress • Expires May 2027   [Delete]│ │
│ │ Created Nov 30, 2025 • Viewed 42 times                  │ │
│ └─────────────────────────────────────────────────────────┘ │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ xyz789... • Progress hidden • Never expires     [Delete]│ │
│ │ Created Dec 1, 2025 • Viewed 7 times                    │ │
│ └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

## Testing

1. **Unit tests**:
   - Token generation
   - Expiration checking
   - PNG generation (visual regression tests with golden files)

2. **Integration tests**:
   - Create share via API
   - Access PNG via public URL
   - Verify expired shares return 404
   - Verify access count increments

3. **Manual testing**:
   - Share to Discord, Twitter, Slack - verify rich previews
   - Test various card states (empty, full, all completed)
   - Test long item text truncation

## Rollout

1. Deploy database migration
2. Deploy backend (share service, PNG generation, public routes)
3. Deploy frontend (share button, modals)
4. Update CSP if needed for any new resources
5. Announce feature to users

## Future Enhancements

- SVG export option (scalable, smaller file size)
- Custom themes/colors for shared images
- QR code on PNG for easy mobile scanning
- Animated GIF showing completion progress over time
- Embeddable widget for blogs/websites
- Share to specific platforms (Twitter, Facebook buttons)

## Dependencies

Add to `go.mod`:
```
github.com/fogleman/gg v1.3.0
github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0
golang.org/x/image v0.14.0
```

Font files (embed in binary or load from disk):
- Include a readable sans-serif font (e.g., Inter, Roboto, or Go's default)
