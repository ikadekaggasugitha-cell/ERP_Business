import { expect, test } from '@playwright/test';

test.describe('Core Finance & Financial Statements Reporting', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/login');
    await page.getByRole('button', { name: /demo/i }).click();
    await expect(page).toHaveURL(/\/$/);
  });

  test('navigates to Journals workspace', async ({ page }) => {
    await page.goto('/journals');
    await expect(page.getByRole('heading').first()).toBeVisible();
  });

  test('switches across P&L, Balance Sheet, and Cash Flow report tabs with CSV export', async ({ page }) => {
    await page.route('**/api/v1/financial-statements/*', async (route) => {
      const url = route.request().url();
      const isBs = url.includes('balance-sheet');
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          statementType: isBs ? 'BALANCE_SHEET' : 'INCOME_STATEMENT',
          currency: 'IDR',
          period: '2026-Q4',
          sections: [
            {
              code: 'MAIN',
              name: isBs ? 'Assets' : 'Revenue',
              items: [{ accountCode: '1100', accountName: 'Cash & Bank', amount: '100000000.0000' }],
              subtotal: '100000000.0000',
            },
          ],
          totals: { netTotal: '100000000.0000' },
          ...(isBs ? { isBalanced: true } : {}),
        }),
      });
    });

    await page.goto('/reports');
    await page.getByRole('tab', { name: /Laba Rugi|Income Statement/i }).click();
    await expect(page.getByRole('link', { name: /CSV/i })).toBeVisible();

    await page.getByRole('tab', { name: /Neraca$|Balance Sheet/i }).click();
    await expect(page.getByText(/Seimbang|Balanced/i)).toBeVisible();

    await page.getByRole('tab', { name: /Arus Kas|Cash Flow/i }).click();
    await expect(page.getByRole('link', { name: /CSV/i })).toBeVisible();
  });
});
