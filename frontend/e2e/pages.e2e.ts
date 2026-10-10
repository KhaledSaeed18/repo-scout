import { expect, test } from '@playwright/test'
import { routes } from './routes'

for (const route of routes) {
  test(`${route.path} renders without errors`, async ({ page }) => {
    const problems: string[] = []
    page.on('pageerror', (err) => problems.push(`page error: ${err.message}`))
    page.on('console', (msg) => {
      if (msg.type() === 'error') problems.push(`console: ${msg.text()}`)
    })
    page.on('response', (res) => {
      if (res.url().includes('/api/') && res.status() >= 400) problems.push(`${res.status()} ${res.url()}`)
    })

    await page.goto(route.path)
    await expect(page.getByRole('heading', { level: 1 })).toHaveText(route.heading)
    // Let queries settle before judging the console.
    await page.waitForLoadState('networkidle')
    await expect(page.getByRole('status').filter({ hasText: /^Loading/ })).toHaveCount(0)
    expect(problems).toEqual([])
  })
}
