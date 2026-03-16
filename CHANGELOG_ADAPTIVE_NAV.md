# Adaptive OPDS Author Navigation - Changelog

## Branch: `feature/adaptive-opds-author-navigation`

### Summary
Implemented intelligent drill-down navigation for OPDS author browsing to handle large libraries (500k+ books) efficiently.

### Changes

#### Database Layer (`internal/db/queries.go`)
- **Added**: `CountAuthorsByPrefix(libraryID, prefix)` - Counts authors matching a prefix
- **Added**: `GetAuthorPrefixCounts(libraryID, prefix, alphabet)` - Returns character→count map for next-level navigation

#### OPDS Handler (`internal/server/handlers_opds.go`)
- **Modified**: `handleOPDSAuthorsByLetterDirect()` - Now implements adaptive navigation
  - Checks author count for given prefix
  - If count > 100: Shows next-level character navigation
  - If count ≤ 100: Shows actual author list
  - Supports both Cyrillic and Latin alphabets
  - Properly URL-encodes Cyrillic characters

#### Documentation
- **Added**: `docs/ADAPTIVE_NAVIGATION.md` - Comprehensive feature documentation
  - How it works
  - Configuration options
  - Performance considerations
  - Testing guidelines
  - Future enhancements

### Technical Details

**Threshold**: 100 authors (configurable in code)

**Navigation Flow**:
```
Single Letter → Count Check
  ├─ > 100 authors → Show 2-letter prefixes → Count Check
  │                    ├─ > 100 authors → Show 3-letter prefixes → ...
  │                    └─ ≤ 100 authors → Show authors
  └─ ≤ 100 authors → Show authors
```

**Performance**:
- Uses existing `idx_author_name` index
- Prefix queries are index-friendly (`LIKE 'prefix%'`)
- Only non-zero counts included in navigation

### Testing Recommendations

1. **Test with your 500k+ book library**:
   ```bash
   # Rebuild and restart the stack
   cd /mnt/hostgit/biblio-suite/biblio-hub
   ./scripts/rebuild_stack.sh
   ```

2. **Test OPDS navigation**:
   - Open OPDS feed in a reader (e.g., FBReader, KOReader)
   - Navigate to Authors
   - Click on a letter with many authors (e.g., "А" or "S")
   - Verify it shows prefixes instead of huge author list
   - Continue drilling down until you see authors

3. **Test both alphabets**:
   - Cyrillic: А, Б, В, ...
   - Latin: A, B, C, ...

4. **Test URL encoding**:
   - Verify Cyrillic characters work correctly in URLs
   - Check browser/OPDS client compatibility

### Deployment

**No database migrations required** - uses existing schema and indexes.

**Steps**:
1. Merge branch to main (or deploy from feature branch)
2. Rebuild Docker images
3. Restart services
4. Test OPDS feed

### Future Enhancements

- [ ] Make threshold configurable via config file
- [ ] Add caching for prefix counts
- [ ] Apply same approach to series navigation
- [ ] Add metrics/logging for navigation patterns
- [ ] Consider pagination as alternative for very large lists

### Commits

1. `feat: implement adaptive OPDS author navigation` - Core implementation
2. `docs: add adaptive navigation documentation` - Documentation

### Related Issues

Addresses the challenge of browsing authors in libraries with 500k+ books where traditional single-letter navigation becomes unwieldy.
