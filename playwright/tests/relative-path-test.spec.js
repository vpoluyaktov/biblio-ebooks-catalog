const { test, expect } = require('@playwright/test');

test('relative path test', async ({ page }) => {
  console.log('Testing relative path /catalog...');
  
  try {
    const response = await page.goto('/catalog', { 
      waitUntil: 'domcontentloaded',
      timeout: 10000 
    });
    console.log('Response status:', response.status());
    console.log('Response URL:', response.url());
    expect(response.status()).toBeLessThan(500);
  } catch (error) {
    console.error('Connection error:', error.message);
    throw error;
  }
});
