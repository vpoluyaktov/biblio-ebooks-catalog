# Playwright Tests for OPDS Server

This directory contains end-to-end tests for the OPDS Server using Playwright.

## Setup

Install dependencies:

```bash
cd playwright
npm install
npx playwright install chromium
```

## Running Tests

Run all tests:
```bash
npm test
```

Run tests with UI:
```bash
npm run test:ui
```

Run tests in headed mode (see browser):
```bash
npm run test:headed
```

## Test Structure

- `tests/` - Test files
  - `keycloak-auth.spec.js` - Keycloak authentication flow tests

## Configuration

- `playwright.config.js` - Playwright configuration
- `package.json` - Node.js dependencies and scripts

## Test Coverage

### Keycloak Authentication Tests
- Redirect to Keycloak login
- Successful authentication with Keycloak
- Access to OPDS dashboard after login
- Admin features access verification
