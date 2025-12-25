const { test, expect } = require('@playwright/test');
const { loginWithCredentials } = require('./helpers');

test('seeded cards render with progress and FREE space', async ({ page }) => {
  await loginWithCredentials(page, 'alice@test.com', 'Password1');

  const card2025 = page.locator('.dashboard-card-preview').filter({
    has: page.locator('.year-badge', { hasText: '2025' }),
  });
  await expect(card2025.first()).toBeVisible();

  await card2025.first().locator('a').first().click();

  await expect(page.locator('.finalized-card-view')).toBeVisible();
  await expect(page.locator('.progress-text')).toContainText('6/24 completed');
  await expect(page.locator('.bingo-cell--free')).toHaveCount(1);
});

test('friend card reactions can be added', async ({ page }) => {
  await loginWithCredentials(page, 'bob@test.com', 'Password1');

  await page.goto('/#friends');
  await expect(page.getByRole('heading', { name: 'Friends', level: 2 })).toBeVisible();

  await expect(page.locator('#friends-list')).toContainText('alice');
  const aliceRow = page.locator('#friends-list .friend-item').filter({ hasText: 'alice' });
  await aliceRow.first().getByRole('link', { name: 'View Card' }).click();

  await expect(page.locator('.finalized-card-view')).toBeVisible();

  await page.locator('.bingo-cell--completed').first().click();
  await expect(page.getByRole('heading', { name: 'Completed Goal' })).toBeVisible();
  if (await page.locator('.reaction-badge').count() === 0) {
    await page.locator('.emoji-btn').first().click();
  }
  await expect(page.locator('.reaction-badge').first()).toBeVisible();
});

test('archived cards show archived view', async ({ page }) => {
  await loginWithCredentials(page, 'alice@test.com', 'Password1');

  const card2024 = () => page.locator('.dashboard-card-preview').filter({
    has: page.locator('.year-badge', { hasText: '2024' }),
  });

  await card2024().locator('.dashboard-card-checkbox').check();
  await expect(page.locator('#selected-count')).toContainText('1 selected');
  await page.getByRole('button', { name: 'Actions' }).click();
  await page.locator('.dropdown-menu .dropdown-item').filter({ hasText: /^\s*Archive\s*$/ }).click();
  await expect(card2024().locator('.archive-badge')).toBeVisible();

  await card2024().locator('a').first().click();
  await expect(page.locator('.bingo-grid--archive')).toBeVisible();
  await expect(page.locator('.archive-badge')).toBeVisible();
});
