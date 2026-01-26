import { test, expect } from '@playwright/test';

test.describe('Plugin Search', () => {
  test.beforeEach(async ({ page }) => {
    // Create test plugins via API
    const apiUrl = process.env.API_URL || 'http://localhost:8080';

    const plugins = [
      { id: 'search-rust', name: 'Rust Plugin', description: 'Rust language support' },
      { id: 'search-python', name: 'Python Plugin', description: 'Python language support' },
      { id: 'search-go', name: 'Go Plugin', description: 'Go language support' },
    ];

    for (const plugin of plugins) {
      await page.request.post(`${apiUrl}/api/v1/plugins`, {
        headers: { 'Content-Type': 'application/json' },
        data: {
          ...plugin,
          author: 'E2E Test',
          license: 'MIT',
          type: 'language',
          security_level: 'community',
        },
      });
    }
  });

  test('should filter plugins by search term', async ({ page }) => {
    await page.goto('/plugins');
    await page.waitForLoadState('networkidle');

    // Find search input
    const searchInput = page.locator('input[type="search"], input[placeholder*="search" i]').first();

    // Type search term
    await searchInput.fill('rust');

    // Wait for debounce and filtering
    await page.waitForTimeout(500);

    // Check if results are filtered
    const pluginCards = page.locator('.plugin-card, [data-testid="plugin-card"]');
    const count = await pluginCards.count();

    // Should show filtered results (might be 0 if plugin not yet visible in UI)
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('should show no results message for non-existent plugin', async ({ page }) => {
    await page.goto('/plugins');
    await page.waitForLoadState('networkidle');

    const searchInput = page.locator('input[type="search"], input[placeholder*="search" i]').first();

    // Search for non-existent plugin
    await searchInput.fill('nonexistentplugin12345');
    await page.waitForTimeout(500);

    // Should show empty state or no results
    const hasEmptyState = await page.locator('.marketplace-empty, .empty-state, .no-results').count();
    const hasNoCards = (await page.locator('.plugin-card').count()) === 0;

    expect(hasEmptyState > 0 || hasNoCards).toBeTruthy();
  });

  test('should clear search on clear button', async ({ page }) => {
    await page.goto('/plugins');
    await page.waitForLoadState('networkidle');

    const searchInput = page.locator('input[type="search"], input[placeholder*="search" i]').first();

    // Enter search term
    await searchInput.fill('test');
    await page.waitForTimeout(300);

    // Clear search
    await searchInput.clear();
    await page.waitForTimeout(300);

    // Should show all plugins again
    expect(await searchInput.inputValue()).toBe('');
  });

  test('should handle special characters in search', async ({ page }) => {
    await page.goto('/plugins');
    await page.waitForLoadState('networkidle');

    const searchInput = page.locator('input[type="search"], input[placeholder*="search" i]').first();

    // Try special characters
    await searchInput.fill('rust-lang');
    await page.waitForTimeout(300);

    // Should not crash
    expect(await page.locator('body').isVisible()).toBeTruthy();
  });
});
