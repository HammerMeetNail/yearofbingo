const { test, expect } = require('@playwright/test');
const { buildUser, register, sendFriendRequest } = require('./helpers');

test('notification rendering escapes usernames', async ({ browser }, testInfo) => {
  const userA = buildUser(testInfo, 'nxssa');
  // Preserve an executable HTML payload while isolating the unique username per project.
  userA.username = `<img src=x onerror=alert(1)>${userA.username}`;
  const userB = buildUser(testInfo, 'nxssb');

  const contextA = await browser.newContext();
  const pageA = await contextA.newPage();
  await register(pageA, userA, { searchable: true });

  const contextB = await browser.newContext();
  const pageB = await contextB.newPage();
  await register(pageB, userB, { searchable: true });

  await sendFriendRequest(pageA, userB.username);

  const message = pageB.locator('.notification-message').first();
  await expect(async () => {
    await pageB.goto('/notifications');
    await expect(message).toBeVisible();
  }).toPass({ timeout: 15000 });
  await expect(message).toContainText(userA.username);
  await expect(message.locator('img')).toHaveCount(0);

  await contextA.close();
  await contextB.close();
});
