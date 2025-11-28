import { test, expect } from '@playwright/test';

test.describe('Basic Application Tests', () => {
  test('homepage loads successfully', async ({ page }) => {
    await page.goto('/');
    
    // Check that the page title or main content loads
    await expect(page).toHaveTitle(/FlightlessSomething|Benchmarks/i);
  });

  test('health endpoint is accessible', async ({ page }) => {
    const response = await page.goto('/health');
    expect(response?.status()).toBe(200);
    
    const body = await response?.text();
    expect(body).toContain('ok');
  });

  test('navigation bar is visible', async ({ page }) => {
    await page.goto('/');
    
    // Check for common navigation elements
    const nav = page.locator('nav, [role="navigation"]');
    await expect(nav).toBeVisible();
  });

  test('benchmarks page loads', async ({ page }) => {
    await page.goto('/');
    
    // Wait for content to load
    await page.waitForLoadState('networkidle');
    
    // Check that we're on a page (it should show some content)
    const body = page.locator('body');
    await expect(body).toBeVisible();
  });

  test('admin login page is accessible', async ({ page }) => {
    await page.goto('/login');
    
    // Should have a login form
    const loginForm = page.locator('form, [role="form"]').first();
    await expect(loginForm).toBeVisible();
  });
});

test.describe('Benchmark List View', () => {
  test('benchmarks list is visible', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    // The page should load without errors
    const mainContent = page.locator('main, .container, #app').first();
    await expect(mainContent).toBeVisible();
  });

  test('search functionality exists', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    // Look for search input
    const searchInput = page.locator('input[type="search"], input[placeholder*="earch" i]');
    if (await searchInput.count() > 0) {
      await expect(searchInput.first()).toBeVisible();
    }
  });
});

test.describe('API Endpoints', () => {
  test('GET /api/benchmarks returns valid JSON', async ({ request }) => {
    const response = await request.get('/api/benchmarks');
    expect(response.ok()).toBeTruthy();
    
    const data = await response.json();
    expect(data).toHaveProperty('benchmarks');
    expect(Array.isArray(data.benchmarks)).toBeTruthy();
  });

  test('GET /health returns ok status', async ({ request }) => {
    const response = await request.get('/health');
    expect(response.ok()).toBeTruthy();
    
    const data = await response.json();
    expect(data).toHaveProperty('status');
    expect(data.status).toBe('ok');
  });
});

test.describe('URL Redirects', () => {
  test('old singular /benchmark/:id URL redirects to /benchmarks/:id', async ({ page }) => {
    // Navigate to the old singular URL format
    await page.goto('/benchmark/1923');
    
    // Wait for redirect to complete
    await page.waitForLoadState('networkidle');
    
    // Check that we were redirected to the new plural URL
    expect(page.url()).toContain('/benchmarks/1923');
  });
  
  test('old singular /benchmark/:id with different ID redirects correctly', async ({ page }) => {
    // Test with a different benchmark ID to ensure the parameter is preserved
    await page.goto('/benchmark/42');
    await page.waitForLoadState('networkidle');
    
    // Verify the ID parameter is correctly passed to the new URL
    expect(page.url()).toContain('/benchmarks/42');
  });
});
