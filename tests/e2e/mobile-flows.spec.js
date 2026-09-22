const { test, expect } = require('@playwright/test');
const { buildUser, register } = require('./helpers');

// Locator.tap uses the browser's native touchscreen input in both mobile projects.
async function tap(locator) {
  await expect(locator).toBeVisible();
  await locator.tap();
}

async function expectNoHorizontalOverflow(page) {
  await expect.poll(() => page.evaluate(() => {
    const root = document.documentElement;
    return Math.max(root.scrollWidth, document.body.scrollWidth) - root.clientWidth;
  })).toBeLessThanOrEqual(1);
}

async function createMobileCard(page, testInfo) {
  await register(page, buildUser(testInfo, 'mobile'));
  await createConfiguredCard(page);
}

async function createConfiguredCard(page) {
  // The dashboard modal exposes grid size; the authenticated /create form does not.
  await page.goto('/dashboard');
  await tap(page.locator('[data-action="show-create-card-modal"]'));
  await expect(page.locator('#modal-title')).toHaveText('Create New Card');
  await page.fill('#modal-card-title', 'Mobile goals');
  await page.selectOption('#modal-card-grid-size', '3');
  await tap(page.locator('#modal-overlay').getByRole('button', { name: 'Create Card', exact: true }));
  await expect(page.locator('#item-input')).toBeVisible();
}

async function addGoal(page, text) {
  await page.fill('#item-input', text);
  await tap(page.locator('#add-btn'));
  await expect(page.locator('#bingo-grid').getByText(text, { exact: true })).toBeVisible();
}

test.beforeEach(async ({ isMobile, hasTouch }) => {
  test.skip(!isMobile || !hasTouch, 'requires a mobile touchscreen project');
});

test('native taps navigate the mobile menu and complete a manually edited card', async ({ page }, testInfo) => {
  await page.goto('/');
  const menu = page.getByRole('button', { name: 'Toggle menu' });
  await tap(menu);
  await expect(menu).toHaveAttribute('aria-expanded', 'true');
  await tap(page.locator('.nav-menu').getByRole('link', { name: 'FAQ', exact: true }));
  await expect(page).toHaveURL(/\/faq$/);
  await expect(menu).toHaveAttribute('aria-expanded', 'false');
  await expectNoHorizontalOverflow(page);

  await createMobileCard(page, testInfo);
  await expectNoHorizontalOverflow(page);
  await addGoal(page, 'Walk outside');
  const goal = page.locator('#bingo-grid .bingo-cell[data-item-id]').first();
  await tap(goal);
  await expect(page.locator('#modal-title')).toHaveText('Edit Goal');
  await expectNoHorizontalOverflow(page);
  await page.locator('textarea[id^="edit-item-content-"]').fill('Walk by the river');
  await tap(page.getByRole('button', { name: 'Save', exact: true }));
  await expect(goal).toContainText('Walk by the river');
  await tap(page.locator('#fill-empty-btn'));
  const finalize = page.locator('.editor-actions').getByRole('button', { name: /Finalize Card/i });
  await expect(finalize).toBeEnabled();
  await tap(finalize);
  await expect(page.locator('#modal-title')).toContainText('Finalize');
  await expectNoHorizontalOverflow(page);
  await tap(page.locator('#modal-overlay').getByRole('button', { name: 'Finalize Card', exact: true }));
  await expect(page.locator('.finalized-card-view')).toBeVisible();
  await expectNoHorizontalOverflow(page);
  await tap(page.locator('#bingo-grid .bingo-cell:not(.bingo-cell--free)').first());
  await tap(page.getByRole('button', { name: 'Mark Complete', exact: true }));
  await expect(page.locator('.progress-text')).toContainText('1/8 completed');
  await tap(page.locator('#bingo-grid .bingo-cell--completed:not(.bingo-cell--free)').first());
  await tap(page.getByRole('button', { name: 'Mark Incomplete', exact: true }));
  await expect(page.locator('.progress-text')).toContainText('0/8 completed');
});

test('mobile AI entry points follow the server feature switch', async ({ page }, testInfo) => {
  const aiRequests = [];
  page.on('request', request => {
    if (request.url().includes('/api/ai/')) aiRequests.push(request.url());
  });
  await register(page, buildUser(testInfo, 'mobileai'));
  const enabled = process.env.FEATURE_AI_ENABLED !== 'false';
  await expect(page.locator('body')).toHaveAttribute('data-ai-enabled', String(enabled));
  const wizard = page.getByRole('button', { name: /Generate with AI Wizard/i });
  if (enabled) {
    await tap(wizard);
    await expect(page.locator('#modal-title')).toContainText('AI Goal Wizard');
    await expectNoHorizontalOverflow(page);
    await tap(page.locator('#modal-close'));
  } else {
    await expect(wizard).toHaveCount(0);
    await expect(page.locator('script[src*="ai-wizard"]')).toHaveCount(0);
  }
  await createConfiguredCard(page);
  await addGoal(page, 'Read a book');
  await tap(page.locator('#bingo-grid .bingo-cell[data-item-id]').first());
  await expect(page.locator('#modal-title')).toHaveText('Edit Goal');
  if (enabled) {
    await tap(page.locator('#ai-refine-generate'));
    const suggestion = page.locator('#ai-refine-results [data-ai-suggestion="0"]');
    await expect(suggestion).toBeVisible();
    const text = (await suggestion.innerText()).trim();
    await tap(suggestion);
    await expect(page.locator('textarea[id^="edit-item-content-"]')).toHaveValue(text);
  } else {
    await expect(page.locator('[data-action^="ai-"]')).toHaveCount(0);
    expect(aiRequests).toEqual([]);
  }
  await expectNoHorizontalOverflow(page);
});

// Chromium drives real native touch sequences through CDP. WebKit has no native
// long-press API in Playwright: its drag coverage dispatches DOM touch events to
// exercise the application's handler semantics (native taps are tested above).
async function touchDriver(page, browserName, source) {
  const session = browserName === 'chromium' ? await page.context().newCDPSession(page) : null;
  const target = await source.elementHandle();
  return {
    async send(type, point) {
      if (session) {
        await session.send('Input.dispatchTouchEvent', {
          type,
          touchPoints: type === 'touchEnd' || type === 'touchCancel' ? [] : [{ ...point, id: 1 }],
        });
      } else {
        await target.evaluate((element, { type, point }) => {
          // WebKit does not expose a constructible Touch. Supply the touch-list
          // fields consumed by the handlers on a synthetic DOM event instead.
          const touch = { identifier: 1, target: element, clientX: point.x, clientY: point.y };
          const ended = type === 'touchEnd' || type === 'touchCancel';
          const event = new Event(type.toLowerCase(), { bubbles: true, cancelable: true });
          Object.assign(event, {
            touches: ended ? [] : [touch], targetTouches: ended ? [] : [touch], changedTouches: [touch],
          });
          element.dispatchEvent(event);
        }, { type, point });
      }
    },
    async dispose() {
      await target.dispose();
      if (session) await session.detach();
    },
  };
}

async function center(locator) {
  const box = await locator.boundingBox();
  expect(box).not.toBeNull();
  return { x: box.x + box.width / 2, y: box.y + box.height / 2 };
}

test('long press reorders, rejects FREE targets, and cancels cleanly (native Chromium; DOM touch WebKit)', async ({ page, browserName }, testInfo) => {
  await createMobileCard(page, testInfo);
  await addGoal(page, 'Move me');
  const cell = position => page.locator(`#bingo-grid .bingo-cell[data-position="${position}"]`);
  const sourcePosition = await page.locator('#bingo-grid .bingo-cell[data-item-id]').first().getAttribute('data-position');
  const targetPosition = await page.locator('#bingo-grid .bingo-cell--empty').first().getAttribute('data-position');
  const freePosition = await page.locator('#bingo-grid .bingo-cell--free').getAttribute('data-position');
  const swaps = [];
  page.on('request', request => {
    if (request.method() === 'POST' && request.url().includes('/swap')) swaps.push(request.url());
  });

  async function gesture(from, to, { cancel = false } = {}) {
    await page.locator('#bingo-grid').scrollIntoViewIfNeeded();
    const start = await center(cell(from));
    const end = await center(cell(to));
    const driver = await touchDriver(page, browserName, cell(from));
    let active = false;
    try {
      await driver.send('touchStart', start);
      active = true;
      await expect(cell(from)).toHaveClass(/bingo-cell--dragging/);
      await expect(page.locator('.bingo-cell--drag-clone')).toHaveCount(1);
      await driver.send('touchMove', end);
      await driver.send(cancel ? 'touchCancel' : 'touchEnd', end);
      active = false;
      await expect(page.locator('.bingo-cell--drag-clone, .bingo-cell--dragging, .bingo-cell--drag-over')).toHaveCount(0);
    } finally {
      try {
        if (active) await driver.send('touchCancel', end);
      } finally {
        await driver.dispose();
      }
    }
  }

  await gesture(sourcePosition, targetPosition);
  await expect(cell(targetPosition)).toContainText('Move me');
  await expect(cell(sourcePosition)).toHaveClass(/bingo-cell--empty/);
  expect(swaps).toHaveLength(1);
  await gesture(targetPosition, freePosition);
  await expect(cell(targetPosition)).toContainText('Move me');
  await expect(cell(freePosition)).toHaveClass(/bingo-cell--free/);
  await gesture(targetPosition, sourcePosition, { cancel: true });
  await expect(cell(targetPosition)).toContainText('Move me');
  await expect(cell(sourcePosition)).toHaveClass(/bingo-cell--empty/);
  expect(swaps).toHaveLength(1);
  await page.reload();
  await expect(cell(targetPosition)).toContainText('Move me');
  await expect(cell(freePosition)).toHaveClass(/bingo-cell--free/);
  await expect(cell(sourcePosition)).toHaveClass(/bingo-cell--empty/);
});
