const { test } = require('@playwright/test');

const token =
  'eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMDAwMCIsImlhdCI6MTc4MzMyNzQ4MSwiZXhwIjoxNzgzMzM0NjgxfQ.yipVew0rTVL4ZHJ4RqZhU4JQ-Pgm66VNLpTNmuMeI6Q';

test('workflow debug page can run without null usage reduce error', async ({ page }) => {
  const errors = [];
  const failedRequests = [];
  const responses = [];

  page.on('console', (msg) => {
    if (['error', 'warning'].includes(msg.type())) {
      errors.push(`[console:${msg.type()}] ${msg.text()}`);
    }
  });
  page.on('pageerror', (error) => {
    errors.push(`[pageerror] ${error.message}\n${error.stack || ''}`);
  });
  page.on('requestfailed', (request) => {
    failedRequests.push(`${request.method()} ${request.url()} ${request.failure()?.errorText}`);
  });
  page.on('response', (response) => {
    const url = response.url();
    if (url.includes('/console/v1/apps') || url.includes('/console/v1/accounts')) {
      responses.push(`${response.status()} ${url}`);
    }
  });

  await page.addInitScript((sessionToken) => {
    window.localStorage.setItem(
      'data-prefers-session',
      JSON.stringify({
        access_token: sessionToken,
        refresh_token: sessionToken,
        expires_time: Date.now() + 60 * 60 * 1000,
      }),
    );
  }, token);

  const response = await page.goto('http://localhost:8000/app/workflow/2073243283097182209', {
    waitUntil: 'domcontentloaded',
    timeout: 60000,
  });
  await page.waitForLoadState('networkidle', { timeout: 30000 }).catch(() => {});
  await page.screenshot({ path: '../output/playwright/workflow-debug-page.png', fullPage: true });

  const bodyText = await page.locator('body').innerText().catch((error) => error.message);
  const buttons = await page
    .locator('button')
    .evaluateAll((nodes) =>
      nodes
        .map((node) => node.innerText || node.getAttribute('aria-label') || node.getAttribute('title') || '')
        .filter(Boolean)
        .slice(0, 40),
    )
    .catch(() => []);

  console.log(
    JSON.stringify(
      {
        status: response?.status(),
        title: await page.title(),
        url: page.url(),
        bodyPreview: bodyText.slice(0, 1600),
        buttons,
        responses,
        failedRequests,
        errors,
      },
      null,
      2,
    ),
  );
});
