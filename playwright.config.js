const { defineConfig, devices } = require('@playwright/test');

const isHeadless = process.env.PLAYWRIGHT_HEADLESS !== 'false';
const baseURL = process.env.PLAYWRIGHT_BASE_URL || 'http://localhost:8080';
const outputDir = process.env.PLAYWRIGHT_OUTPUT_DIR || 'test-results';
const reportDir = process.env.PLAYWRIGHT_REPORT_DIR || 'playwright-report';

module.exports = defineConfig({
  testDir: 'tests/e2e',
  timeout: 60000,
  expect: {
    timeout: 10000,
  },
  fullyParallel: false,
  workers: 1,
  outputDir,
  reporter: [
    ['list'],
    ['html', { open: 'never', outputFolder: reportDir }],
  ],
  use: {
    baseURL,
    headless: isHeadless,
    actionTimeout: 10000,
    navigationTimeout: 30000,
    acceptDownloads: true,
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    trace: 'retain-on-failure',
  },
  projects: [
    {
      name: 'firefox',
      use: { ...devices['Desktop Firefox'] },
    },
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'webkit',
      use: { ...devices['Desktop Safari'] },
    },
  ],
});
