const { test, expect } = require('@playwright/test');
const { buildUser, register, expectToast } = require('./helpers');

test('AI wizard generates goals and creates a card', async ({ page }, testInfo) => {
  const user = buildUser(testInfo, 'aiwiz');
  await register(page, user);

  await page.getByRole('button', { name: /Generate with AI Wizard/i }).click();
  await expect(page.locator('#modal-title')).toContainText('AI Goal Wizard');

  await page.check('input[name="difficulty"][value="medium"]');
  await page.check('input[name="budget"][value="free"]');
  await page.fill('#ai-focus', 'Local adventures');
  await page.evaluate(() => {
    document.getElementById('ai-wizard-form')?.requestSubmit();
  });

  await expect(page.locator('#modal-title')).toContainText('Review Your Goals');
  await expect(page.locator('.ai-goal-input')).toHaveCount(24);

  await page.locator('#modal-overlay').getByRole('button', { name: 'Create Card →' }).click();
  await expectToast(page, 'AI Card Created!');
  await expect(page.locator('#item-input')).toBeVisible();
  await expect(page.locator('.bingo-cell:not(.bingo-cell--free)')).toHaveCount(24);
  await expect(page.locator('#bingo-grid')).toContainText('Sunrise Walk');
});

test('AI wizard requires verification after free generations are used', async ({ page }, testInfo) => {
  const user = buildUser(testInfo, 'aigate');
  await register(page, user);

  await page.getByRole('button', { name: /Generate with AI Wizard/i }).click();
  await expect(page.locator('#modal-title')).toContainText('AI Goal Wizard');

  for (let i = 0; i < 5; i += 1) {
    await page.check('input[name="difficulty"][value="medium"]');
    await page.check('input[name="budget"][value="free"]');
    await page.fill('#ai-focus', `Hiking ${i}`);
    await page.evaluate(() => {
      document.getElementById('ai-wizard-form')?.requestSubmit();
    });
    await expect(page.locator('#modal-title')).toContainText('Review Your Goals');

    const startOver = page.getByRole('button', { name: 'Start Over' });
    await startOver.scrollIntoViewIfNeeded();
    if (i < 4) {
      await startOver.click();
      await expect(page.locator('#modal-title')).toContainText('AI Goal Wizard');
    } else {
      await startOver.click();
    }
  }

  await expect(page.locator('#modal-title')).toContainText('Verify Email Required');
});
