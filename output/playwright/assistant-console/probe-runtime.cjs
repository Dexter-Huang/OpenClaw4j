async (page) => {
  const records = {
    console: [],
    pageErrors: [],
    exceptions: [],
    htmlScripts: [],
    failedRequests: [],
  };
  const client = await page.context().newCDPSession(page);

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
    const failure = request.failure();
    records.failedRequests.push({
      url: request.url(),
      failure: failure ? failure.errorText : '',
    });
  });

  page.on('response', async (response) => {
    const url = response.url();
    const contentType = response.headers()['content-type'] || '';
    if (url.includes('.js') && contentType.includes('text/html')) {
      records.htmlScripts.push({
        url,
        status: response.status(),
        contentType,
        preview: (await response.text()).slice(0, 160),
      });
    }
  });

  client.on('Runtime.exceptionThrown', ({ exceptionDetails }) => {
    records.exceptions.push({
      text: exceptionDetails.text,
      url: exceptionDetails.url,
      lineNumber: exceptionDetails.lineNumber,
      columnNumber: exceptionDetails.columnNumber,
      description:
        exceptionDetails.exception && exceptionDetails.exception.description,
    });
  });

  await client.send('Runtime.enable');
  await page.reload({ waitUntil: 'domcontentloaded' });
  await page
    .getByText('API Configuration', { exact: false })
    .waitFor({ timeout: 60000 });
  await page.waitForTimeout(5000);

  return records;
}
