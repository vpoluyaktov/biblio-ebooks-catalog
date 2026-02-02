# Biblio Auth Integration - Work in Progress

This document tracks the integration of Biblio Auth to replace Keycloak/OIDC authentication.

## Completed Work

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

## Remaining Work

### 1. Server Handler Updates
The following files need updates to work with the new auth manager:
- `internal/server/server.go` - Update middleware calls
- `internal/server/handlers_auth.go` - Update auth endpoints
- `internal/server/handlers_web.go` - Update web UI auth checks
- `main.go` - Remove `cfg.Auth.Mode` references

### 2. Middleware Updates
- `internal/auth/middleware.go` needs Biblio Auth session validation
- Add middleware to validate Biblio Auth tokens from cookies
- Update context to store Biblio Auth user info

### 3. Documentation
- Remove Keycloak references from `Specification.md`
- Remove Keycloak references from `README.md`
- Remove Keycloak references from `playwright/README.md`
- Update authentication mode documentation

### 4. Testing
- Update Playwright tests to use Biblio Auth instead of Keycloak
- Test Basic Auth for OPDS (should still work)
- Test web UI login flow with Biblio Auth
- Test admin features

## Architecture

### Authentication Flow

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
# Biblio Auth URL (internal Docker network)
BIBLIO_AUTH_URL=http://biblio-auth:80

# Basic Auth for OPDS (optional)
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
