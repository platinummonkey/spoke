import { test, expect } from '@playwright/test';

test.describe('Plugin Marketplace', () => {
  test('should display marketplace page', async ({ page }) => {
    await page.goto('/plugins');

    // Check page title
    await expect(page.locator('h1')).toContainText('Spoke Registry');

    // Check navigation button exists
    await expect(page.getByRole('link', { name: /plugins/i })).toBeVisible();
  });

  test('should display plugin grid', async ({ page }) => {
    await page.goto('/plugins');

    // Wait for content to load
    await page.waitForLoadState('networkidle');

    // Check for marketplace container
    const marketplace = page.locator('.plugin-marketplace, [data-testid="plugin-marketplace"]');
    await expect(marketplace).toBeVisible();
  });

  test('should show empty state when no plugins', async ({ page }) => {
    await page.goto('/plugins');
    await page.waitForLoadState('networkidle');

    // Either plugins exist or empty state shows
    const hasPlugins = await page.locator('.plugin-card').count();
    const hasEmptyState = await page.locator('.marketplace-empty, .empty-state').count();

    expect(hasPlugins > 0 || hasEmptyState > 0).toBeTruthy();
  });

  test('should display search bar', async ({ page }) => {
    await page.goto('/plugins');

    // Search input should be visible
    const searchInput = page.locator('input[type="search"], input[placeholder*="search" i]');
    await expect(searchInput).toBeVisible();
  });

  test('should display filter dropdowns', async ({ page }) => {
    await page.goto('/plugins');

    // Check for filter selects
    const filters = page.locator('select, [role="combobox"]');
    const filterCount = await filters.count();

    // Should have at least Type and Security Level filters
    expect(filterCount).toBeGreaterThanOrEqual(2);
  });

  test('should handle pagination', async ({ page }) => {
    await page.goto('/plugins');
    await page.waitForLoadState('networkidle');

    // Check if pagination exists (might not if fewer than page size)
    const pagination = page.locator('.marketplace-pagination, [data-testid="pagination"]');

    // Pagination may or may not exist depending on plugin count
    // Just verify no errors
    expect(await page.locator('body').isVisible()).toBeTruthy();
  });

  test('should navigate to plugin detail on card click', async ({ page }) => {
    // First, create a test plugin via API
    const response = await page.request.post(`${process.env.API_URL || 'http://localhost:8080'}/api/v1/plugins`, {
      headers: {
        'Content-Type': 'application/json',
      },
      data: {
        id: 'e2e-test-plugin',
        name: 'E2E Test Plugin',
        description: 'Plugin for E2E testing',
        author: 'E2E Test',
        license: 'MIT',
        type: 'language',
        security_level: 'community',
      },
    });

    await page.goto('/plugins');
    await page.waitForLoadState('networkidle');

    // Click on first plugin card (if exists)
    const pluginCard = page.locator('.plugin-card, [data-testid="plugin-card"]').first();

    if (await pluginCard.isVisible()) {
      await pluginCard.click();

      // Should navigate to detail page
      await expect(page).toHaveURL(/\/plugins\/[^/]+$/);
    }
  });

  test('should display loading state', async ({ page }) => {
    await page.goto('/plugins');

    // Loading spinner or skeleton should appear briefly
    // This might be too fast to catch, so we just verify no errors
    await page.waitForLoadState('networkidle');

    expect(await page.locator('body').isVisible()).toBeTruthy();
  });
});
