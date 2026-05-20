import { expect, test } from '@playwright/test';
import { mockApi } from './fixtures';

test.beforeEach(async ({ page }) => {
  await mockApi(page);
});

test('unauthenticated dashboard redirects to login', async ({ page }) => {
  await page.goto('/dashboard');

  await expect(page).toHaveURL(/\/login$/);
  await expect(page.getByLabel('Username')).toBeVisible();
});

test('login stores token and shows dashboard', async ({ page }) => {
  await page.goto('/login');
  await page.getByLabel('Username').fill('alice');
  await page.getByLabel('Password').fill('secret');
  await page.getByRole('button', { name: 'Sign in' }).click();

  await expect(page).toHaveURL(/\/dashboard$/);
  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible();
  await expect(page.getByText('Remaining Quota')).toBeVisible();
  await expect(page.evaluate(() => localStorage.getItem('token'))).resolves.toBe('test-user-token');
});

test('admin token enables Options nav', async ({ page }) => {
  await page.goto('/login');
  await page.evaluate(() => {
    localStorage.setItem('token', 'test-user-token');
    localStorage.setItem('adminToken', 'test-admin-token');
  });
  await page.goto('/dashboard');

  await expect(page.getByRole('link', { name: 'Options' })).toBeVisible();
});

test('admin options renders core settings', async ({ page }) => {
  await page.goto('/login');
  await page.evaluate(() => {
    localStorage.setItem('token', 'test-user-token');
    localStorage.setItem('adminToken', 'test-admin-token');
  });
  await page.goto('/admin/options');

  await expect(page.getByRole('heading', { name: 'System Options' })).toBeVisible();
  await expect(page.getByText('Core Settings')).toBeVisible();
  await expect(page.getByText('Registration enabled')).toBeVisible();
});

test('admin users sends sort and filter params', async ({ page }) => {
  const requests: string[] = [];
  await page.route('**/api/user**', async (route) => {
    if (route.request().method() === 'GET') {
      requests.push(route.request().url());
      await route.fulfill({ json: { success: true, data: [{ id: '1', username: 'alice', status: 1, group: 'default' }] } });
      return;
    }
    await route.continue();
  });

  await page.goto('/login');
  await page.evaluate(() => {
    localStorage.setItem('token', 'test-user-token');
    localStorage.setItem('adminToken', 'test-admin-token');
  });
  await page.goto('/admin/users');
  await page.getByLabel('Filter users by status').selectOption('1');
  await page.getByRole('button', { name: /sort by username/i }).click();

  expect(
    requests.some((url) => url.includes('status=1') && url.includes('sort=username') && url.includes('order=asc')),
  ).toBe(true);
});
