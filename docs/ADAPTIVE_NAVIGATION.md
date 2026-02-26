# Adaptive OPDS Author Navigation

## Overview

The adaptive navigation feature provides intelligent drill-down navigation for browsing authors in OPDS feeds, especially useful for large libraries with hundreds of thousands of books.

## How It Works

### Traditional Approach (Before)
- Show single-letter navigation (A, B, C, ...)
- Clicking a letter shows ALL authors starting with that letter
- Problem: With 500k+ books, a single letter might have thousands of authors

### Adaptive Approach (After)
1. **Initial Level**: Show single-letter navigation (А, Б, В, ... / A, B, C, ...)
2. **Smart Drill-Down**: When clicking a letter/prefix:
   - Count authors matching that prefix
   - If count > threshold (100): Show next-level prefixes (AA, AB, AC, ...)
   - If count ≤ threshold: Show actual author list
3. **Recursive**: Process continues until author count is manageable

### Example Navigation Flow

For a library with 500k books:

```
Root → Authors
  ├─ А (15,234 authors) → Drill down
  │   ├─ АБ (1,456 authors) → Drill down
  │   │   ├─ АБР (89 authors) → Show authors
  │   │   └─ АБУ (45 authors) → Show authors
  │   └─ АВ (234 authors) → Drill down
  │       ├─ АВГ (67 authors) → Show authors
  │       └─ АВД (23 authors) → Show authors
  └─ Б (8,123 authors) → Drill down
      └─ ...
```

## Configuration

### Threshold
The threshold is currently hardcoded at **100 authors** in `handlers_opds.go`:

```go
const threshold = 100
```

You can adjust this value based on your preferences:
- **Lower threshold (50)**: More drill-down levels, smaller lists
- **Higher threshold (200)**: Fewer drill-down levels, longer lists

### Supported Alphabets
- **Cyrillic**: АБВГДЕЖЗИЙКЛМНОПРСТУФХЦЧШЩЭЮЯ
- **Latin**: ABCDEFGHIJKLMNOPQRSTUVWXYZ

The system automatically detects which alphabet to use based on the first character of the prefix.

## Database Functions

### `CountAuthorsByPrefix(libraryID, prefix)`
Counts authors whose `last_name` starts with the given prefix.

```sql
SELECT COUNT(DISTINCT a.id) FROM author a
WHERE a.library_id = ? AND a.last_name LIKE 'prefix%'
```

### `GetAuthorPrefixCounts(libraryID, prefix, alphabet)`
Returns a map of character → count for all possible next characters.

Example:
```go
counts := {
  "А": 1234,
  "Б": 567,
  "В": 890,
  // ... only characters with authors
}
```

## Performance Considerations

### Database Indexes
The existing schema already includes an optimal index for prefix queries:

```sql
CREATE INDEX idx_author_name ON author(last_name, first_name);
```

This composite index supports efficient prefix matching on `last_name`.

### Query Optimization
- Prefix matching uses `LIKE 'prefix%'` which is index-friendly
- Only characters with authors are included in navigation
- Counts are calculated on-demand (could be cached in future)

## Future Enhancements

### Potential Improvements
1. **Configurable Threshold**: Make threshold configurable per library
2. **Caching**: Cache prefix counts to reduce database queries
3. **Smart Alphabet**: Dynamically determine alphabet based on actual data
4. **Mixed Alphabets**: Handle authors with names in different scripts
5. **Series Navigation**: Apply same adaptive approach to series browsing

### Alternative Approaches
- **Pagination**: Instead of drill-down, use pagination with large lists
- **Search-First**: Encourage search over browsing for very large libraries
- **Hybrid**: Combine adaptive navigation with search suggestions

## Testing

### Test with Different Library Sizes
- **Small (<1k authors)**: Should show authors directly
- **Medium (1k-10k)**: Should drill down 1-2 levels
- **Large (100k+)**: Should drill down 2-4 levels
- **Very Large (500k+)**: Should drill down 3-5 levels

### Test Cases
1. Navigate to single letter with few authors → Shows authors
2. Navigate to single letter with many authors → Shows prefixes
3. Navigate through multiple drill-down levels → Eventually shows authors
4. Test both Cyrillic and Latin alphabets
5. Test URL encoding for Cyrillic characters

## Implementation Details

### URL Structure
```
/opds/{libID}/authors           → Root alphabet
/opds/{libID}/authors/А         → Single letter (may drill down)
/opds/{libID}/authors/АБ        → Two letters (may drill down)
/opds/{libID}/authors/АБР       → Three letters (shows authors)
```

### Feed Structure
Navigation feed (drill-down):
```xml
<entry>
  <title>АБ (1,456)</title>
  <link rel="subsection" href="/opds/1/authors/АБ"/>
</entry>
```

Acquisition feed (authors):
```xml
<entry>
  <title>Абрамов, Александр (42)</title>
  <link rel="subsection" href="/opds/1/author/12345"/>
</entry>
```

## Migration Notes

### Backward Compatibility
- Existing OPDS clients will work without changes
- URLs remain the same structure
- Only behavior changes (drill-down vs direct list)

### Deployment
1. Deploy updated code
2. No database migrations required
3. Existing indexes should be sufficient
4. Monitor query performance on first use

## References

- Similar implementation in Calibre OPDS server
- OPDS 1.2 specification: https://specs.opds.io/opds-1.2
- SQLite LIKE optimization: https://www.sqlite.org/optoverview.html#like_opt
