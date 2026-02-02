# Biblio Auth Integration Status

This document tracks the integration of Biblio Auth to replace Keycloak/OIDC authentication in biblio-ebooks-catalog.

**Overall Status:** 90% Complete - Core integration done, one routing issue remaining

## ✅ Completed Work

### 1. Configuration Updates
- ✅ Removed `OIDCConfig` from `internal/config/config.go`
- ✅ Added `BiblioAuthConfig` with `URL` field
- ✅ Updated environment variable loading to use `BIBLIO_AUTH_URL`
- ✅ Removed `Mode` field from `AuthConfig` (no longer needed)

### 2. Dependencies
- ✅ Removed OIDC dependencies from `go.mod`:
  - `github.com/coreos/go-oidc/v3`
  - `github.com/go-jose/go-jose/v4`
  - `golang.org/x/oauth2`

### 3. Auth Implementation
- ✅ Created `internal/auth/biblioauth.go` - Biblio Auth client
  - `ValidateSession()` - Validates auth tokens with Biblio Auth
  - `GetLoginURL()` - Returns login redirect URL
  - `GetLogoutURL()` - Returns logout URL
  - `IsAdmin()` - Checks admin privileges
- ✅ Simplified `internal/auth/manager.go`
  - Removed OIDC mode complexity
  - Uses Biblio Auth for web UI authentication
  - Keeps internal auth for Basic Auth (OPDS)
- ✅ Removed `internal/auth/oidc.go`
- ✅ Removed `internal/server/handlers_oidc.go`

## 🔧 Remaining Work

### Critical Issue: API Endpoint Path Routing (404 Error)

**Problem:** Catalog is calling `http://biblio-auth:80/api/validate` but getting 404

**Root Cause:** The catalog's `BiblioAuthClient` uses `baseURL` set to `http://biblio-auth:80` and appends `/api/validate`. However, Biblio Auth's API endpoints are served at `/auth/api/*` through nginx, not directly at `/api/*`.

**Current Behavior:**
```
Catalog → http://biblio-auth:80/api/validate (direct service call)
Result: 404 Not Found
```

**Expected Behavior:**
```
Catalog → http://nginx-gateway:80/auth/api/validate (through nginx)
OR
Catalog → http://biblio-auth:80/auth/api/validate (if Biblio Auth serves both paths)
Result: 200 OK with user validation
```

**Solution Options:**

1. **Option A: Route through nginx (Recommended)**
   - Update `BIBLIO_AUTH_URL` in stack.yaml to `http://nginx-gateway:80/auth`
   - Catalog will call `http://nginx-gateway:80/auth/api/validate`
   - Pros: Uses existing nginx routing, consistent with external access
   - Cons: Extra hop through nginx for internal calls

2. **Option B: Update Biblio Auth to serve both paths**
   - Add routes in Biblio Auth to serve `/api/*` in addition to `/auth/api/*`
   - Catalog continues calling `http://biblio-auth:80/api/validate`
   - Pros: Direct service-to-service communication
   - Cons: Duplicate route definitions in Biblio Auth

3. **Option C: Update base path in config**
   - Change Biblio Auth base path handling to include `/auth` prefix
   - Update environment variable to `BIBLIO_AUTH_URL=http://biblio-auth:80/auth`
   - Pros: Clean configuration
   - Cons: Requires Biblio Auth code changes

**Recommended Fix:** Option A - Update stack.yaml to use nginx gateway URL

### 2. Testing & Verification
- [ ] Fix the 404 routing issue
- [ ] Test complete login flow (login → redirect → authenticated session)
- [ ] Verify session persistence across page reloads
- [ ] Test logout flow
- [ ] Test Basic Auth for OPDS (should still work with internal auth)
- [ ] Test admin features with Biblio Auth groups

### 3. Documentation Updates
- [ ] Remove Keycloak references from `Specification.md`
- [ ] Remove Keycloak references from `README.md`
- [ ] Update Playwright tests documentation
- [ ] Document the three auth modes clearly

## Architecture

### Authentication Modes

The catalog supports three authentication modes via `AUTH_MODE` environment variable:

#### 1. `AUTH_MODE=internal` (Standalone)
- Catalog manages its own users in SQLite database
- Displays its own login screen
- Used for standalone deployments outside BiblioHub
- OPDS Basic Auth uses internal user database

#### 2. `AUTH_MODE=oidc` (Legacy - Keycloak)
- Integrates with Keycloak or other OIDC providers
- OAuth2 Authorization Code flow for web UI
- OPDS Basic Auth validated via Keycloak ROPC
- **Being deprecated** - replaced by biblio-auth mode

#### 3. `AUTH_MODE=biblio-auth` (New - BiblioHub Stack)
- Integrates with Biblio Auth service
- Web UI redirects to Biblio Auth for login
- JWT token validation via Biblio Auth API
- OPDS Basic Auth uses internal user database (for e-reader compatibility)
- **This is the target mode** for BiblioHub stack deployment

### Authentication Flow (biblio-auth mode)

**Web UI:**
1. User accesses catalog → Redirected to Biblio Auth login
2. Biblio Auth validates credentials → Issues JWT token as cookie
3. Catalog validates token with Biblio Auth `/api/validate` endpoint
4. User info stored in request context

**OPDS/E-readers:**
1. E-reader sends HTTP Basic Auth
2. Catalog validates against internal user database
3. Session created for authenticated user

### Environment Variables

```bash
# Authentication mode: internal, oidc, or biblio-auth
AUTH_MODE=biblio-auth

# Biblio Auth URL (internal Docker network, used when AUTH_MODE=biblio-auth)
BIBLIO_AUTH_URL=http://biblio-auth:80

# Basic Auth for OPDS (optional, used in all modes)
OPDS_AUTH_ENABLED=true
OPDS_AUTH_USER=admin
OPDS_AUTH_PASSWORD=secret
```

## Next Steps

1. Fix compilation errors in server handlers
2. Implement Biblio Auth session middleware
3. Update web UI JavaScript to use Biblio Auth endpoints
4. Remove all Keycloak references from documentation
5. Test full integration
6. Update Playwright tests

## Notes

- Basic Auth for OPDS clients is preserved (uses internal user database)
- Web UI authentication now uses Biblio Auth exclusively
- No more dual-mode (internal/oidc) complexity
- Simpler configuration and deployment
