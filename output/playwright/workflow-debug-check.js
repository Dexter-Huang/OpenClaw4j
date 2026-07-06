const { chromium } = require('playwright');

const backendBaseUrl = 'http://127.0.0.1:9004';

async function login() {
  const response = await fetch(`${backendBaseUrl}/console/v1/auth/login`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      username: 'saa',
      password: '123456',
    }),
  });

  if (!response.ok) {
    throw new Error(`Login failed: ${response.status} ${await response.text()}`);
  }

  const result = await response.json();
  return result.data;
}

(async () => {
  const session = await login();
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
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

  await page.addInitScript((sessionData) => {
    window.localStorage.setItem(
      'data-prefers-session',
      JSON.stringify({
        access_token: sessionData.access_token,
        refresh_token: sessionData.refresh_token,
        expires_time: sessionData.expires_in * 1000,
      }),
    );
  }, session);

  const response = await page.goto('http://localhost:8000/app/workflow/2073243283097182209', {
    waitUntil: 'domcontentloaded',
    timeout: 60000,
  });
  await page.waitForLoadState('networkidle', { timeout: 30000 }).catch(() => {});

  await page.getByRole('button', { name: /^Run$/ }).click({ timeout: 15000 }).catch((error) => {
    errors.push(`[action] click Run failed: ${error.message}`);
  });
  await page.waitForTimeout(1500);

  const runPanelText = await page.locator('body').innerText().catch(() => '');
  const startButton = page
    .getByRole('button', { name: /开始测试|Start testing|Run/ })
    .last();
  if (/开始测试|Start testing/.test(runPanelText)) {
    const queryInput = page
      .locator('textarea, input')
      .filter({ hasNotText: /^$/ })
      .first();
    await queryInput.fill('1+1=?').catch(() => {});
    await startButton.click({ timeout: 10000 }).catch((error) => {
      errors.push(`[action] click start test failed: ${error.message}`);
    });
    await page.waitForTimeout(5000);
  }

  await page.screenshot({ path: 'output/playwright/workflow-debug-page.png', fullPage: true });

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

  await browser.close();
})();
