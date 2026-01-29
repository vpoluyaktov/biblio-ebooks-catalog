const { test, expect } = require('@playwright/test');

test.describe('OPDS Server Keycloak Authentication', () => {
  const KEYCLOAK_USERNAME = 'testadmin';
  const KEYCLOAK_PASSWORD = 'admin123';

  test('should redirect to Keycloak login when accessing OPDS catalog', async ({ page }) => {
    // Navigate to OPDS catalog
    await page.goto('http://localhost:9900/catalog/');
    
    // Should be redirected to Keycloak login page
    await expect(page).toHaveURL(/\/auth\/realms\/biblio\/protocol\/openid-connect\/auth/);
    
    // Verify Keycloak login page elements
    await expect(page.locator('text=Sign in to your account')).toBeVisible();
    await expect(page.locator('input[id="username"]')).toBeVisible();
    await expect(page.locator('input[id="password"]')).toBeVisible();
  });

  test('should successfully authenticate with Keycloak and access OPDS catalog', async ({ page }) => {
    // Navigate to OPDS catalog
    await page.goto('http://localhost:9900/catalog/');
    
    // Wait for redirect to Keycloak
    await page.waitForURL(/\/auth\/realms\/biblio\/protocol\/openid-connect\/auth/);
    
    // Fill in Keycloak login form
    await page.fill('input[id="username"]', KEYCLOAK_USERNAME);
    await page.fill('input[id="password"]', KEYCLOAK_PASSWORD);
    
    // Submit login form by pressing Enter
    await page.press('input[id="password"]', 'Enter');
    
    // Should be redirected back to OPDS catalog
    await page.waitForURL(/\/catalog/);
    
    // Verify we're authenticated
    const authInfo = await page.evaluate(async () => {
      const response = await fetch('/catalog/api/auth/info');
      return response.json();
    });
    
    expect(authInfo.authenticated).toBe(true);
    expect(authInfo.mode).toBe('keycloak');
    expect(authInfo.user.username).toBe(KEYCLOAK_USERNAME);
    expect(authInfo.user.role).toBe('admin');
  });

  test('should show OPDS dashboard after successful login', async ({ page }) => {
    // Navigate to OPDS catalog
    await page.goto('http://localhost:9900/catalog/');
    
    // Wait for redirect to Keycloak
    await page.waitForURL(/\/auth\/realms\/biblio\/protocol\/openid-connect\/auth/);
    
    // Login
    await page.fill('input[id="username"]', KEYCLOAK_USERNAME);
    await page.fill('input[id="password"]', KEYCLOAK_PASSWORD);
    await page.press('input[id="password"]', 'Enter');
    
    // Wait for redirect back to OPDS
    await page.waitForURL(/\/catalog/);
    
    // Wait for dashboard to load
    await page.waitForSelector('text=OPDS Server', { timeout: 10000 });
    
    // Verify dashboard elements
    await expect(page.locator('text=Dashboard')).toBeVisible();
    await expect(page.locator('text=testadmin')).toBeVisible();
    
    // Verify admin can see Import Library button
    await expect(page.locator('text=Import Library')).toBeVisible();
  });

  test('should have access to admin features after Keycloak login', async ({ page }) => {
    // Navigate and login
    await page.goto('http://localhost:9900/catalog/');
    await page.waitForURL(/\/auth\/realms\/biblio\/protocol\/openid-connect\/auth/);
    await page.fill('input[id="username"]', KEYCLOAK_USERNAME);
    await page.fill('input[id="password"]', KEYCLOAK_PASSWORD);
    await page.press('input[id="password"]', 'Enter');
    await page.waitForURL(/\/catalog/);
    
    // Wait for page to load
    await page.waitForSelector('text=OPDS Server', { timeout: 10000 });
    
    // Try to access libraries API endpoint (admin only)
    const librariesResponse = await page.evaluate(async () => {
      const response = await fetch('/catalog/api/libraries');
      return {
        status: response.status,
        ok: response.ok
      };
    });
    
    expect(librariesResponse.ok).toBe(true);
    expect(librariesResponse.status).toBe(200);
  });
});
