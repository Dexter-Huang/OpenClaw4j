async (page) => {
  const records = {
    console: [],
    interactions: [],
  };

  page.on('console', (msg) => {
    const text = msg.text();
    if (
      text.includes('findDOMNode') ||
      text.includes('[antd: Tooltip]') ||
      text.includes('validateDOMNesting')
    ) {
      records.console.push({
        type: msg.type(),
        text,
        location: msg.location(),
      });
    }
  });

  await page.reload({ waitUntil: 'domcontentloaded' });
  await page
    .getByText('API Configuration', { exact: false })
    .waitFor({ timeout: 60000 });
  await page.waitForTimeout(2000);

  const locators = [
    page.locator('button'),
    page.locator('[role="button"]'),
    page.locator('.spark-icon'),
    page.locator('.anticon'),
  ];

  for (const locator of locators) {
    const count = Math.min(await locator.count(), 80);
    for (let index = 0; index < count; index += 1) {
      const item = locator.nth(index);
      if (!(await item.isVisible().catch(() => false))) {
        continue;
      }

      const label = await item
        .evaluate((node) => {
          const element = node;
          return {
            tagName: element.tagName,
            text: element.textContent?.trim().slice(0, 80),
            className:
              typeof element.className === 'string' ? element.className : '',
            ariaLabel: element.getAttribute('aria-label'),
            title: element.getAttribute('title'),
          };
        })
        .catch(() => null);

      records.interactions.push({ action: 'hover', label });
      await item.hover({ timeout: 1000 }).catch(() => undefined);
      await page.waitForTimeout(80);
    }
  }

  return records;
}
