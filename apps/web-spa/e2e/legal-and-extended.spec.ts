import { expect, test } from '@playwright/test';

const EXTENDED_ROUTES = [
  '/fixed-assets',
  '/dimensions',
  '/budgets',
  '/financial-close',
  '/consolidation',
  '/projects',
  '/assets',
  '/maintenance',
  '/crm',
  '/localization',
];

test.describe('Public Legal Trio & 10 Extended Service Workspaces', () => {
  test('public legal routes render without authentication', async ({ page }) => {
    await page.goto('/privacy');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible();

    await page.goto('/terms');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible();

    await page.goto('/delete-account');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
  });

  test('all 10 extended service routes render after login', async ({ page }) => {
    await page.goto('/login');
    await page.getByRole('button', { name: /demo/i }).click();
    await expect(page).toHaveURL(/\/$/);

    for (const route of EXTENDED_ROUTES) {
      await page.goto(route);
      await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
    }
  });
});
