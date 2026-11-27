import { expect, test } from '@playwright/test';

test.describe('Authentication, 3-Tier Admin RBAC & Org Switcher', () => {
  test('renders 3-tier login personas and 4-node organization selector', async ({ page }) => {
    await page.goto('/login');
    await expect(page.getByText(/30 Microservices/i)).toBeVisible();

    const orgTrigger = page.locator('#orgCode');
    await expect(orgTrigger).toBeVisible();
    await orgTrigger.click();
    await expect(page.getByRole('option', { name: /HLD-01/i })).toBeVisible();
    await expect(page.getByRole('option', { name: /SUB-PRO/i })).toBeVisible();
    await expect(page.getByRole('option', { name: /SUB-LOG/i })).toBeVisible();
    await expect(page.getByRole('option', { name: /BRC-JKT/i })).toBeVisible();
  });

  test('offline demo login displays Head Office Admin tier badge and org switcher', async ({ page }) => {
    await page.goto('/login');
    await page.getByRole('button', { name: /admin\.ho/i }).click();
    await page.getByRole('button', { name: /demo/i }).click();

    await expect(page).toHaveURL(/\/$/);
    await page.getByRole('button', { name: /org/i }).first().click();
    await expect(page.getByText(/HLD-01/i).first()).toBeVisible();
  });
});
