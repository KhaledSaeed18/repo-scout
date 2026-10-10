import { expect, test } from '@playwright/test'
import { routes } from './routes'

for (const colorScheme of ['light', 'dark'] as const) {
  test.describe(`${colorScheme} theme at 390px`, () => {
    test.use({ colorScheme })

    for (const route of routes) {
      test(`${route.path} fits the screen`, async ({ page }) => {
        await page.goto(route.path)
        await expect(page.getByRole('heading', { level: 1 })).toHaveText(route.heading)
        await page.waitForLoadState('networkidle')
        expect(await page.evaluate(() => document.documentElement.classList.contains('dark'))).toBe(colorScheme === 'dark')
        // Wide tables scroll inside their own container; the page itself must not.
        const overflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth)
        expect(overflow).toBeLessThanOrEqual(0)
      })
    }
  })
}
