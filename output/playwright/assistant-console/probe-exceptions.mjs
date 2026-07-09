import { chromium } from 'playwright';
import { writeFile } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const targetUrl =
  process.env.OPENCLAW_ASSISTANT_URL ??
  'http://localhost:8000/app/assistant/2072641584112357378';

const loginResponse = await fetch(
  'http://127.0.0.1:9004/console/v1/auth/login',
  {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({
      username: process.env.OPENCLAW_USER ?? 'saa',
      password: process.env.OPENCLAW_PASSWORD ?? '123456',
    }),
  },
);

if (!loginResponse.ok) {
  throw new Error(`Login failed: ${loginResponse.status}`);
}

const loginPayload = await loginResponse.json();
const sessionData = {
  access_token: loginPayload.data.access_token,
  refresh_token: loginPayload.data.refresh_token,
  expires_time: loginPayload.data.expires_in * 1000,
};

const browser = await chromium.launch({ headless: true });
const context = await browser.newContext();
const page = await context.newPage();
const cdp = await context.newCDPSession(page);

const records = {
  console: [],
  pageErrors: [],
  exceptions: [],
  failedRequests: [],
  htmlScripts: [],
};

page.on('console', (msg) => {
  records.console.push({
    type: msg.type(),
    text: msg.text(),
    location: msg.location(),
  });
});

page.on('pageerror', (error) => {
  records.pageErrors.push({
    message: error.message,
    stack: error.stack,
  });
});

page.on('requestfailed', (request) => {
  records.failedRequests.push({
    url: request.url(),
    method: request.method(),
    failure: request.failure()?.errorText,
  });
});

page.on('response', async (response) => {
  const url = response.url();
  const contentType = response.headers()['content-type'] ?? '';
  if (!url.includes('.js') || !contentType.includes('text/html')) {
    return;
  }

  records.htmlScripts.push({
    url,
    status: response.status(),
    contentType,
    preview: (await response.text()).slice(0, 160),
  });
});

cdp.on('Runtime.exceptionThrown', ({ exceptionDetails }) => {
  records.exceptions.push({
    text: exceptionDetails.text,
    url: exceptionDetails.url,
    lineNumber: exceptionDetails.lineNumber,
    columnNumber: exceptionDetails.columnNumber,
    description: exceptionDetails.exception?.description,
  });
});

try {
  await cdp.send('Runtime.enable');
  await page.goto('http://localhost:8000/', { waitUntil: 'domcontentloaded' });
  await page.evaluate((value) => {
    localStorage.setItem('data-prefers-session', JSON.stringify(value));
  }, sessionData);
  await page.goto(targetUrl, { waitUntil: 'domcontentloaded' });
  await page
    .getByText('API Configuration', { exact: false })
    .waitFor({ timeout: 60000 });
  await page.waitForTimeout(5000);
} finally {
  await writeFile(
    join(__dirname, 'exception-probe.json'),
    JSON.stringify(records, null, 2),
    'utf8',
  );
  await browser.close();
}

console.log(JSON.stringify(records, null, 2));
