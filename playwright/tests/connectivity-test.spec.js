const { test, expect } = require('@playwright/test');

test('basic connectivity test', async ({ page }) => {
  console.log('Testing connectivity to localhost:9900...');
  
  try {
    const response = await page.goto('http://localhost:9900/catalog/', { 
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
