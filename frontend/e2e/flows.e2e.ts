import { expect, test } from '@playwright/test'

test('search finds text inside files', async ({ page }) => {
  await page.goto('/search')
  await page.getByLabel('Search for').fill('greeting')
  await page.getByRole('button', { name: 'Search', exact: true }).click()
  await expect(page.getByText('util/util.go').first()).toBeVisible()
})

test('hotspots rank the file that keeps changing', async ({ page }) => {
  await page.goto('/hotspots')
  await page.getByRole('combobox', { name: 'Count changes from' }).click()
  await page.getByRole('option', { name: 'All history' }).click()
  const first = page.getByRole('row').nth(1)
  await expect(first).toContainText('util/util.go')
  await first.getByRole('link').click()
  await expect(page).toHaveURL(/\/files\?file=util%2Futil\.go/)
  await expect(page.getByRole('heading', { name: 'util.go' })).toBeVisible()
})

test('knowledge shows both authors', async ({ page }) => {
  await page.goto('/knowledge')
  await expect(page.getByText('bus factor')).toBeVisible()
  const authors = page.locator('section', { has: page.getByRole('heading', { name: 'Main authors' }) })
  await expect(authors.getByText('ana', { exact: true })).toBeVisible()
  await expect(authors.getByText('bo', { exact: true })).toBeVisible()
})

test('duplicates list the repeated block', async ({ page }) => {
  await page.goto('/duplicates')
  await expect(page.getByRole('link', { name: 'dup/a.go' })).toBeVisible()
  await expect(page.getByRole('link', { name: 'dup/b.go' })).toBeVisible()
})

test('scanning again keeps the results on screen', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByRole('link', { name: /lines of code/ })).toBeVisible()
  await page.getByRole('button', { name: 'Scan again' }).click()
  await expect(page.getByRole('button', { name: 'Scan again' })).toBeEnabled({ timeout: 30_000 })
  await expect(page.getByRole('link', { name: /lines of code/ })).toBeVisible()
})
