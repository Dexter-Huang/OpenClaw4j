const { chromium } = require('playwright');

const backendBaseUrl = 'http://127.0.0.1:9004';
const pageUrl = 'http://localhost:8000/app/workflow/2073243283097182209';

async function login() {
  const response = await fetch(`${backendBaseUrl}/console/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username: 'saa', password: '123456' }),
  });

  if (!response.ok) {
    throw new Error(`Login failed: ${response.status} ${await response.text()}`);
  }

  return (await response.json()).data;
}

(async () => {
  const session = await login();
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  const cdp = await page.context().newCDPSession(page);

  const exceptions = [];
  const htmlResources = [];
  const failedRequests = [];
  const responseSummary = [];

  await cdp.send('Runtime.enable');
  cdp.on('Runtime.exceptionThrown', (event) => {
    const detail = event.exceptionDetails || {};
    exceptions.push({
      text: detail.text,
      url: detail.url,
      lineNumber: detail.lineNumber,
      columnNumber: detail.columnNumber,
      exception: detail.exception?.description || detail.exception?.value,
      stack: detail.stackTrace?.callFrames?.slice(0, 5),
    });
  });

  page.on('requestfailed', (request) => {
    failedRequests.push({
      method: request.method(),
      url: request.url(),
      resourceType: request.resourceType(),
      errorText: request.failure()?.errorText,
    });
  });

  page.on('response', async (response) => {
    const request = response.request();
    const contentType = response.headers()['content-type'] || '';
    const item = {
      status: response.status(),
      url: response.url(),
      resourceType: request.resourceType(),
      contentType,
    };

    if (
      response.status() >= 400 ||
      (contentType.includes('text/html') &&
        !['document', 'xhr', 'fetch'].includes(request.resourceType()))
    ) {
      responseSummary.push(item);
    }

    if (
      contentType.includes('text/html') &&
      !['document', 'xhr', 'fetch'].includes(request.resourceType())
    ) {
      const body = await response.text().catch((error) => error.message);
      htmlResources.push({
        ...item,
        bodyStart: body.slice(0, 220),
      });
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

  const response = await page.goto(pageUrl, {
    waitUntil: 'domcontentloaded',
    timeout: 60000,
  });
  await page.waitForLoadState('networkidle', { timeout: 30000 }).catch(() => {});
  await page.waitForTimeout(3000);

  console.log(
    JSON.stringify(
      {
        pageStatus: response?.status(),
        pageTitle: await page.title(),
        finalUrl: page.url(),
        exceptions,
        htmlResources,
        failedRequests,
        responseSummary,
      },
      null,
      2,
    ),
  );

  await browser.close();
})();
