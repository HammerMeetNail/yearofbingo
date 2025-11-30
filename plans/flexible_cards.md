# Flexible Cards Plan

## Overview

Allow users to create bingo cards with custom dimensions and header words beyond the traditional 5x5 "BINGO" format. Supports use cases like monthly cards (3-letter month abbreviations), weekly challenges, and custom themes.

## User Stories

1. As a user, I want to create a monthly card with "DEC" as the header (3x3 or 3x5)
2. As a user, I want to create a weekly card with "WEEK" (4 columns)
3. As a user, I want to choose whether my card has a FREE space
4. As a user, I want to place the FREE space wherever I want
5. As a user, I want my existing BINGO cards to continue working unchanged

## Design Decisions

- **Grid shape**: Flexible - rows and columns independent
- **Header word**: Required, determines column count
- **FREE space**: Optional, user-placed (not auto-centered)
- **Migration strategy**: Keep classic BINGO cards separate, add new "custom" card type
- **Backwards compatibility**: All existing features continue working for classic cards

## Dimension Constraints

### Columns (determined by header word length)

| Min | Max | Rationale |
|-----|-----|-----------|
| 2 | 10 | 2 is minimum viable grid; 10 keeps UI manageable |

**Header word validation**:
- 2-10 characters
- Letters only (A-Z)
- Displayed uppercase
- Common examples: GO, WIN, LUCK, GOAL, BINGO, MONDAY, CHALLENGE

### Rows

| Min | Max | Rationale |
|-----|-----|-----------|
| 2 | 10 | Match column limits for consistency |

### Total Cells

| Min | Max | Rationale |
|-----|-----|-----------|
| 4 | 100 | 2x2 minimum; 10x10 maximum |

**Item count**: Total cells minus FREE space (if enabled)
- Minimum: 3 items (2x2 with FREE)
- Maximum: 100 items (10x10 no FREE)

### Recommended Configurations

| Use Case | Header | Rows | Items | Notes |
|----------|--------|------|-------|-------|
| Monthly | DEC, JAN, etc. | 3-4 | 8-11 | Quick monthly goals |
| Weekly | WEEK | 4-5 | 15-19 | Week-long challenges |
| Classic | BINGO | 5 | 24 | Traditional annual card |
| Quarterly | GOAL | 4-5 | 15-19 | Quarterly objectives |
| Extended | CHALLENGE | 5-6 | 25-59 | Ambitious long-term |

## Database Schema Changes

### Option A: Add columns to existing table (recommended)

```sql
-- Add to bingo_cards table
ALTER TABLE bingo_cards ADD COLUMN card_type VARCHAR(10) DEFAULT 'classic';
ALTER TABLE bingo_cards ADD COLUMN header_word VARCHAR(10) DEFAULT 'BINGO';
ALTER TABLE bingo_cards ADD COLUMN row_count INTEGER DEFAULT 5;
ALTER TABLE bingo_cards ADD COLUMN has_free_space BOOLEAN DEFAULT true;
ALTER TABLE bingo_cards ADD COLUMN free_space_position INTEGER DEFAULT 12;

-- Constraints
ALTER TABLE bingo_cards ADD CONSTRAINT valid_card_type
    CHECK (card_type IN ('classic', 'custom'));
ALTER TABLE bingo_cards ADD CONSTRAINT valid_header_word
    CHECK (header_word ~ '^[A-Z]{2,10}$');
ALTER TABLE bingo_cards ADD CONSTRAINT valid_row_count
    CHECK (row_count >= 2 AND row_count <= 10);
ALTER TABLE bingo_cards ADD CONSTRAINT valid_free_position
    CHECK (free_space_position IS NULL OR free_space_position >= 0);
```

**Column count**: Derived from `LENGTH(header_word)`

**Classic cards**: Existing cards get `card_type='classic'`, which enforces:
- `header_word='BINGO'`
- `row_count=5`
- `has_free_space=true`
- `free_space_position=12`

### Position Calculation

Current 5x5: positions 0-24, row-major order
```
 0  1  2  3  4
 5  6  7  8  9
10 11 12 13 14
15 16 17 18 19
20 21 22 23 24
```

Flexible NxM (N cols, M rows): positions 0 to (N*M - 1)
```
Example: 3x4 (DEC with 4 rows)
 0  1  2
 3  4  5
 6  7  8
 9 10 11
```

**Position helpers**:
```go
func (c *BingoCard) Columns() int {
    return len(c.HeaderWord)
}

func (c *BingoCard) TotalCells() int {
    return c.Columns() * c.RowCount
}

func (c *BingoCard) ItemCount() int {
    if c.HasFreeSpace {
        return c.TotalCells() - 1
    }
    return c.TotalCells()
}

func (c *BingoCard) PositionToRowCol(pos int) (row, col int) {
    cols := c.Columns()
    return pos / cols, pos % cols
}

func (c *BingoCard) RowColToPosition(row, col int) int {
    return row * c.Columns() + col
}
```

## Bingo Detection Changes

### Current Algorithm (5x5 hardcoded)

```go
// Hardcoded patterns for 5x5
var bingoPatterns = [][]int{
    {0, 1, 2, 3, 4},     // Row 0
    {5, 6, 7, 8, 9},     // Row 1
    // ... etc
    {0, 5, 10, 15, 20},  // Column 0
    // ... etc
    {0, 6, 12, 18, 24},  // Diagonal
    {4, 8, 12, 16, 20},  // Anti-diagonal
}
```

### New Algorithm (dynamic)

```go
func (c *BingoCard) GenerateBingoPatterns() [][]int {
    cols := c.Columns()
    rows := c.RowCount
    patterns := [][]int{}

    // Rows
    for r := 0; r < rows; r++ {
        pattern := make([]int, cols)
        for col := 0; col < cols; col++ {
            pattern[col] = r*cols + col
        }
        patterns = append(patterns, pattern)
    }

    // Columns
    for col := 0; col < cols; col++ {
        pattern := make([]int, rows)
        for r := 0; r < rows; r++ {
            pattern[r] = r*cols + col
        }
        patterns = append(patterns, pattern)
    }

    // Diagonals (only for square grids)
    if cols == rows {
        // Main diagonal
        diag := make([]int, cols)
        for i := 0; i < cols; i++ {
            diag[i] = i*cols + i
        }
        patterns = append(patterns, diag)

        // Anti-diagonal
        antiDiag := make([]int, cols)
        for i := 0; i < cols; i++ {
            antiDiag[i] = i*cols + (cols - 1 - i)
        }
        patterns = append(patterns, antiDiag)
    }

    return patterns
}

func (c *BingoCard) CountBingos(completedPositions map[int]bool) int {
    patterns := c.GenerateBingoPatterns()
    count := 0

    for _, pattern := range patterns {
        bingo := true
        for _, pos := range pattern {
            // FREE space counts as complete
            if c.HasFreeSpace && pos == c.FreeSpacePosition {
                continue
            }
            if !completedPositions[pos] {
                bingo = false
                break
            }
        }
        if bingo {
            count++
        }
    }
    return count
}
```

### Bingo rules by grid shape

| Shape | Rows | Columns | Diagonals | Total Possible |
|-------|------|---------|-----------|----------------|
| Square (NxN) | N | N | 2 | 2N + 2 |
| Rectangular | M | N | 0 | M + N |

**Note**: Diagonals only count for square grids where a true diagonal exists.

## Impact Analysis

### 1. Card Creation UI (`renderCardEditor`)

**Changes needed**:
- New "Create Custom Card" flow separate from classic
- Header word input (validates 2-10 letters)
- Row count selector (2-10)
- FREE space toggle and position picker
- Preview of grid dimensions before creation
- Dynamic grid rendering based on dimensions

**Complexity**: High - significant UI changes

### 2. Card Display (`renderFinalizedCard`)

**Changes needed**:
- Dynamic CSS grid columns (`grid-template-columns: repeat(N, 1fr)`)
- Responsive breakpoints based on column count
- Cell sizing adjustments for readability
- Header row with custom word letters

**Complexity**: Medium - CSS and JS changes

### 3. Suggestions System

**Current**: 80 curated suggestions, designed for 24-item cards

**Impact**:
- Small cards (3-8 items): Suggestions overkill, but still useful
- Large cards (50+ items): May need more suggestions
- "Fill Empty" button needs to respect card size

**Changes needed**:
- Cap random fill at actual item count
- Consider category-based suggestions for themes

**Complexity**: Low

### 4. Statistics & Progress

**Current**: "X/24 Complete", bingo count

**Changes needed**:
- Dynamic denominator: "X/{itemCount} Complete"
- Bingo count uses dynamic patterns
- Percentage calculations unchanged (X/total * 100)

**Complexity**: Low

### 5. Friend Card View

**Changes needed**:
- Render friend's card with their dimensions
- Stats display adapts to their card size
- Works with existing visibility model

**Complexity**: Low - uses same rendering logic

### 6. PNG Generation (plans/png.md)

**Impact**: Significant

**Changes needed**:
- Dynamic image dimensions based on grid size
- Cell size calculations for readability
- Text truncation adjustments for smaller cells
- Stats panel positioning
- May need different aspect ratios

**Considerations**:
- Small grids (2x2): Image could be smaller
- Large grids (10x10): Cells become tiny, text unreadable
- Solution: Set minimum cell size, scale image dimensions

**Complexity**: High

### 7. API Token Access (plans/api.md)

**Changes needed**:
- Card creation accepts new parameters
- Item positions validated against card dimensions
- Response includes card dimensions

**New request format**:
```json
POST /api/cards
{
    "year": 2025,
    "type": "custom",           // or "classic"
    "header_word": "DEC",       // custom only
    "row_count": 4,             // custom only
    "has_free_space": true,     // custom only
    "free_space_position": 4    // custom only, optional
}
```

**Complexity**: Low - additive changes

### 8. Export Functionality

**Changes needed**:
- CSV export includes card dimensions
- Header row uses custom word
- Position mapping accounts for variable grid

**Complexity**: Low

### 9. Drag and Drop

**Changes needed**:
- Works with any grid size
- Swap endpoint unchanged (positions are just numbers)
- Touch handling unchanged

**Complexity**: None - already position-agnostic

### 10. Database Queries

**Changes needed**:
- Position validation in item creation
- Free space position validation
- No hardcoded position references

**Complexity**: Low

### 11. Frontend Bingo Detection (JS)

**Current**: Hardcoded 5x5 patterns in `app.js`

**Changes needed**:
- Port dynamic pattern generation to JavaScript
- Or fetch patterns from server with card data

**Complexity**: Medium

## Migration Strategy

### Phase 0: Preparation

1. Audit all code for hardcoded `5`, `24`, `12`, `25` values
2. Create abstraction layer for card dimensions
3. Add feature flag for custom cards

### Phase 1: Database Migration

```sql
-- Migration: Add flexible card columns
ALTER TABLE bingo_cards ADD COLUMN card_type VARCHAR(10) DEFAULT 'classic';
ALTER TABLE bingo_cards ADD COLUMN header_word VARCHAR(10) DEFAULT 'BINGO';
ALTER TABLE bingo_cards ADD COLUMN row_count INTEGER DEFAULT 5;
ALTER TABLE bingo_cards ADD COLUMN has_free_space BOOLEAN DEFAULT true;
ALTER TABLE bingo_cards ADD COLUMN free_space_position INTEGER DEFAULT 12;

-- Set existing cards as classic
UPDATE bingo_cards SET
    card_type = 'classic',
    header_word = 'BINGO',
    row_count = 5,
    has_free_space = true,
    free_space_position = 12;

-- Add constraints
ALTER TABLE bingo_cards ADD CONSTRAINT valid_card_type
    CHECK (card_type IN ('classic', 'custom'));
```

### Phase 2: Backend Changes

1. Update `BingoCard` model with new fields
2. Add dimension helper methods
3. Update `CardService` for custom card creation
4. Dynamic bingo detection
5. Position validation in item handlers
6. Update card stats calculation

### Phase 3: API Changes

1. Card creation accepts custom parameters
2. Card response includes dimensions
3. Validation for custom card constraints
4. Update API documentation

### Phase 4: Frontend - Card Creation

1. "New Card" flow with type selection
2. Custom card wizard:
   - Header word input
   - Row count slider
   - FREE space toggle and position picker
   - Preview grid
3. Classic card creation unchanged

### Phase 5: Frontend - Card Display

1. Dynamic CSS grid rendering
2. Header word display
3. Responsive breakpoints by size
4. Cell sizing optimization

### Phase 6: Frontend - Bingo Detection

1. Port dynamic pattern algorithm to JS
2. Update progress calculations
3. Update stats display

### Phase 7: PNG Generation

1. Dynamic image sizing
2. Cell size calculations
3. Layout adjustments
4. Test various dimensions

### Phase 8: Testing & Polish

1. Test all grid sizes (edge cases: 2x2, 10x10)
2. Visual regression tests
3. Performance testing (large grids)
4. Accessibility review

## UI Mockups

### Card Type Selection

```
┌─────────────────────────────────────────────────────────────┐
│ Create New Card                                             │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────────────┐   ┌─────────────────────┐         │
│  │    ╔═══╦═══╦═══╗    │   │   ┌───┬───┬───┐    │         │
│  │    ║ B ║ I ║ N ║    │   │   │ ? │ ? │ ? │    │         │
│  │    ╠═══╬═══╬═══╣    │   │   ├───┼───┼───┤    │         │
│  │    ║ G ║ O ║   ║    │   │   │ ? │ ? │ ? │    │         │
│  │    ╚═══╩═══╩═══╝    │   │   └───┴───┴───┘    │         │
│  │                     │   │                     │         │
│  │   Classic BINGO     │   │   Custom Card       │         │
│  │   5x5 grid, 24      │   │   Choose your own   │         │
│  │   items + FREE      │   │   dimensions        │         │
│  │                     │   │                     │         │
│  │      [Create]       │   │      [Create]       │         │
│  └─────────────────────┘   └─────────────────────┘         │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Custom Card Wizard

```
┌─────────────────────────────────────────────────────────────┐
│ Create Custom Card                                        ✕ │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│ Header Word (determines columns)                            │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ DEC                                                     │ │
│ └─────────────────────────────────────────────────────────┘ │
│ 3 columns                                                   │
│                                                             │
│ Number of Rows                                              │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ ◀  [====●=========]  ▶                                 │ │
│ └─────────────────────────────────────────────────────────┘ │
│ 4 rows                                                      │
│                                                             │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ [✓] Include FREE space                                  │ │
│ └─────────────────────────────────────────────────────────┘ │
│                                                             │
│ Preview: 3x4 grid = 11 items + 1 FREE                       │
│                                                             │
│  ┌─────┬─────┬─────┐                                       │
│  │  D  │  E  │  C  │                                       │
│  ├─────┼─────┼─────┤                                       │
│  │     │     │     │                                       │
│  ├─────┼─────┼─────┤                                       │
│  │     │ FREE│     │  ← Click to move FREE space           │
│  ├─────┼─────┼─────┤                                       │
│  │     │     │     │                                       │
│  ├─────┼─────┼─────┤                                       │
│  │     │     │     │                                       │
│  └─────┴─────┴─────┘                                       │
│                                                             │
│                            [Cancel] [Create Card]           │
└─────────────────────────────────────────────────────────────┘
```

### FREE Space Placement

```
Click any cell to place the FREE space:

  ┌─────┬─────┬─────┐
  │  D  │  E  │  C  │
  ├─────┼─────┼─────┤
  │  ○  │  ○  │  ○  │
  ├─────┼─────┼─────┤
  │  ○  │ ★FREE│  ○  │  ← Current position
  ├─────┼─────┼─────┤
  │  ○  │  ○  │  ○  │
  ├─────┼─────┼─────┤
  │  ○  │  ○  │  ○  │
  └─────┴─────┴─────┘
```

## CSS Considerations

### Dynamic Grid

```css
.bingo-grid {
    display: grid;
    /* Set dynamically based on column count */
    grid-template-columns: repeat(var(--cols), 1fr);
    gap: 4px;
}

.bingo-grid[data-cols="3"] { --cols: 3; }
.bingo-grid[data-cols="4"] { --cols: 4; }
.bingo-grid[data-cols="5"] { --cols: 5; }
/* ... up to 10 */
```

### Responsive Cell Sizing

```css
/* Base cell size by column count */
.bingo-grid[data-cols="3"] .bingo-cell { min-height: 120px; }
.bingo-grid[data-cols="5"] .bingo-cell { min-height: 100px; }
.bingo-grid[data-cols="7"] .bingo-cell { min-height: 80px; }
.bingo-grid[data-cols="10"] .bingo-cell { min-height: 60px; font-size: 0.75rem; }

/* Text truncation scales with cell size */
.bingo-grid[data-cols="10"] .bingo-cell {
    -webkit-line-clamp: 2;  /* Less text for tiny cells */
}
```

## Testing Strategy

### Unit Tests

1. Dimension validation (min/max)
2. Position calculations (row/col conversions)
3. Bingo pattern generation for various sizes
4. Bingo counting with/without FREE space
5. Item count calculations

### Integration Tests

1. Create custom card via API
2. Add items at various positions
3. FREE space handling
4. Bingo detection for square grids
5. Bingo detection for rectangular grids

### Visual Regression Tests

1. 2x2 grid rendering
2. 5x5 classic rendering (unchanged)
3. 10x10 large grid rendering
4. Various FREE space positions
5. Mobile responsive layouts

### Edge Cases

1. 2x2 with FREE = only 3 items
2. 10x10 with no FREE = 100 items
3. 1x10 grid (if allowed) - just a row
4. 10x1 grid - just a column
5. Non-square grids with no diagonals

## Security Considerations

1. **Input validation**: Sanitize header word (letters only, uppercase)
2. **Size limits**: Enforce 2-10 for both dimensions
3. **Position validation**: Items can't exceed grid size
4. **FREE space position**: Must be within grid bounds
5. **Classic cards protected**: Can't convert classic to custom via API

## Performance Considerations

1. **Large grids**: 100 cells = 100 potential items
   - Lazy load items if needed
   - Pagination for item lists
2. **Bingo calculation**: O(patterns × items)
   - For 10x10: 22 patterns × 100 positions = 2200 checks
   - Still fast, no optimization needed
3. **PNG generation**: Larger images take longer
   - Consider caching aggressively

## Rollout Plan

1. **Feature flag**: `ENABLE_CUSTOM_CARDS=false` initially
2. **Beta testing**: Enable for select users
3. **Documentation**: Update FAQ, add tutorial
4. **Gradual rollout**: Enable for all users
5. **Monitor**: Track custom card creation rates, grid size distribution

## Future Enhancements

- **Templates**: Pre-defined configurations (Monthly, Weekly, Quarterly)
- **Themes**: Color schemes per card type
- **Collaborative cards**: Multiple users share one card
- **Card duplication**: Copy card structure to new year
- **Import/Export**: JSON format for card structure
- **Header emojis**: Allow emoji headers (🎯🎲🎪)

## Summary

| Aspect | Impact | Effort |
|--------|--------|--------|
| Database | Medium | Low |
| Backend logic | Medium | Medium |
| Card creation UI | High | High |
| Card display UI | Medium | Medium |
| PNG generation | High | High |
| API changes | Low | Low |
| Testing | High | Medium |

**Total estimated effort**: Large feature, recommend phased rollout over multiple releases.

**Recommendation**: Start with Phase 1-3 (backend), then Phase 4-5 (UI), then Phase 7 (PNG) as a follow-up release.
