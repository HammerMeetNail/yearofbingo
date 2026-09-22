const { test, expect } = require('@playwright/test');
const {
  buildUser,
  register,
  createCardFromAuthenticatedCreate,
  fillCardWithSuggestions,
  finalizeCard,
} = require('./helpers');

async function requireAIDisabled(page) {
  test.skip(process.env.FEATURE_AI_ENABLED !== 'false', 'requires FEATURE_AI_ENABLED=false');
  await page.goto('/');
  const enabled = await page.locator('body').getAttribute('data-ai-enabled');
  expect(enabled).toBe('false');
}

test('disabled AI keeps anonymous card creation and suggestions available', async ({ page }) => {
  const aiRequests = [];
  page.on('request', (request) => {
    if (request.url().includes('/api/ai/')) aiRequests.push(request.url());
  });

  await requireAIDisabled(page);
  await page.goto('/create');

  await expect(page.locator('body')).not.toContainText('AI Goal Wizard');
  await expect(page.locator('[data-action^="ai-"]')).toHaveCount(0);
  await expect(page.locator('script[src*="ai-wizard"]')).toHaveCount(0);

  await page.fill('#card-title', 'AI Disabled Card');
  await page.selectOption('#card-grid-size', '3');
  await page.getByRole('button', { name: 'Create Card' }).click();
  await expect(page.locator('#item-input')).toBeVisible();
  await expect(page.locator('.suggestions-panel')).toBeVisible();
  await expect(page.locator('#ai-btn')).toHaveCount(0);
  await expect(page.locator('#ai-fill-empty-btn')).toHaveCount(0);

  await page.fill('#item-input', 'Anonymous manual goal');
  await page.click('#add-btn');
  await expect(page.locator('.bingo-cell[data-item-id]')).toHaveCount(1);
  await fillCardWithSuggestions(page);
  expect(aiRequests).toEqual([]);
});

test('disabled AI preserves authenticated card creation and finalization', async ({ page }, testInfo) => {
  const aiRequests = [];
  page.on('request', (request) => {
    if (request.url().includes('/api/ai/')) aiRequests.push(request.url());
  });

  await requireAIDisabled(page);
  const user = buildUser(testInfo, 'aidisabled');
  await register(page, user);
  await expect(page.locator('[data-action^="ai-"]')).toHaveCount(0);
  await expect(page.locator('script[src*="ai-wizard"]')).toHaveCount(0);

  await createCardFromAuthenticatedCreate(page, { title: 'AI Disabled Auth Card' });
  await expect(page.locator('body')).not.toContainText('AI Goal Wizard');
  await expect(page.locator('#ai-btn')).toHaveCount(0);
  await expect(page.locator('#ai-fill-empty-btn')).toHaveCount(0);
  await fillCardWithSuggestions(page);
  await finalizeCard(page);
  await expect(page.locator('.finalized-card-view')).toBeVisible();
  expect(aiRequests).toEqual([]);
});

test('disabled AI is removed from premium UI and rejects direct AI calls', async ({ page }) => {
  await requireAIDisabled(page);
  await page.goto('/premium');
  await expect(page.getByRole('heading', { name: 'Premium', exact: true })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Templates + rollover' })).toBeVisible();

  await expect(page.locator('body')).not.toContainText('AI Enhancements');
  await expect(page.locator('[data-action^="ai-"]')).toHaveCount(0);
  await expect(page.locator('#premium-ai-status')).toHaveCount(0);

  const csrfResponse = await page.request.get('/api/csrf');
  expect(csrfResponse.ok()).toBeTruthy();
  const csrfPayload = await csrfResponse.json();
  expect(csrfPayload?.token).toBeTruthy();

  const response = await page.request.post('/api/ai/generate', {
    headers: { 'X-CSRF-Token': csrfPayload.token },
    data: {
      category: 'mix',
      focus: '',
      difficulty: 'medium',
      budget: 'free',
      context: '',
      count: 1,
    },
  });
  expect(response.status()).toBe(503);
});
