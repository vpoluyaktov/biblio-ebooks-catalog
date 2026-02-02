# Biblio Catalog - Current Development Status

## Biblio Auth Integration (In Progress)

**Branch:** `feature/biblio-auth-integration`  
**Status:** 95% Complete - Routing issue fixed, testing remaining

### What's Working
- ✅ Frontend redirects to Biblio Auth login page
- ✅ Biblio Auth login/logout functionality
- ✅ JWT token generation and cookie management
- ✅ Catalog validates auth_token cookies
- ✅ Three auth modes supported: internal, oidc (deprecated), biblio-auth
- ✅ Token validation working (404 issue fixed)

### Resolved Issue: 404 on Token Validation

**Problem:** Catalog was calling `http://biblio-auth:80/api/validate` → 404 Not Found

**Root Cause:** Biblio Auth serves APIs at `/auth/api/*` due to `BIBLIO_AUTH_BASE_PATH=/auth`

**Fix Applied:** Updated `BIBLIO_AUTH_URL` in stack.yaml from `http://biblio-auth:80` to `http://biblio-auth:80/auth`

**Details:** See `/home/ubuntu/git/biblio-ebooks-catalog/BIBLIO_AUTH_INTEGRATION.md`

### Next Steps
1. ~~Apply the fix (update stack.yaml BIBLIO_AUTH_URL)~~ ✅ Done
2. ~~Test complete login flow~~ ✅ Done
3. Remove Keycloak references from documentation
4. Update Playwright tests

### Authentication Architecture

**Web UI (biblio-auth mode):**
- User visits catalog → Redirected to Biblio Auth login
- Biblio Auth validates → Issues JWT as `auth_token` cookie
- Catalog validates token via Biblio Auth API
- User context stored in request

**OPDS/E-readers (all modes):**
- HTTP Basic Auth
- Validated against internal user database
- Compatible with e-reader apps

### Environment Variables

```bash
# Auth mode: internal, oidc (deprecated), or biblio-auth
AUTH_MODE=biblio-auth

# Biblio Auth URL (needs fix - should route through nginx)
BIBLIO_AUTH_URL=http://nginx-gateway:80/auth  # Recommended
# Currently: http://biblio-auth:80  # Causes 404

# Basic Auth for OPDS (optional)
OPDS_AUTH_ENABLED=true
OPDS_AUTH_USER=admin
OPDS_AUTH_PASSWORD=secret
```

### Files Modified
- `internal/config/config.go` - Added BiblioAuthConfig
- `internal/auth/biblioauth.go` - Biblio Auth client
- `internal/auth/manager.go` - Simplified auth manager
- `internal/server/auth_helpers.go` - Session validation by mode
- `internal/server/handlers_auth.go` - Auth info endpoint
- `web/static/js/app.js` - Frontend redirect logic
- `go.mod` - Removed OIDC dependencies

### Files Removed
- `internal/auth/oidc.go`
- `internal/server/handlers_oidc.go`
