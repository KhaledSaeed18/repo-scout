// Optional documentation tooling; does not add dependencies to the app.
import { createServer } from 'node:http'
import { createRequire } from 'node:module'
import { execFileSync } from 'node:child_process'
import { readFile, writeFile, mkdir } from 'node:fs/promises'
import { dirname, extname, resolve, relative, sep } from 'node:path'
import { fileURLToPath } from 'node:url'

const here = dirname(fileURLToPath(import.meta.url))
const root = resolve(here, '../..')
const require = createRequire(resolve(root, '.cache/showcase-tools/package.json'))
const { chromium } = require('playwright')
const origin = process.env.SHOWCASE_APP_URL || 'http://localhost:5173'
const api = process.env.SHOWCASE_API_URL || 'http://127.0.0.1:8080'
const repoId = Number(process.env.SHOWCASE_REPO_ID || 1)
const structureRepoId = Number(process.env.SHOWCASE_STRUCTURE_REPO_ID || 2)
const capture = process.argv.includes('--capture')
const browser = await chromium.launch(process.env.SHOWCASE_CHROMIUM
  ? { executablePath: process.env.SHOWCASE_CHROMIUM }
  : { channel: 'chrome' })

try {
  if (capture) {
    const response = await fetch(`${api}/api/repositories/${repoId}`)
    if (!response.ok) throw new Error(`Repository ${repoId}: HTTP ${response.status}`)
    const repo = await response.json()
    if (repo.status !== 'ready' || !repo.commitCount || !repo.fileCount) {
      throw new Error('Finish scanning a repository with real files and Git history before capturing.')
    }
    // The captions describe this particular dataset. Update the template to use another.
    if (!/tanstack\/query(?:\.git)?$/i.test(repo.gitRemote || '')) {
      throw new Error('These captions expect TanStack/query. Update showcase.html for another repository.')
    }
    const settings = await (await fetch(`${api}/api/settings`)).json()
    if (settings.theme !== 'system') throw new Error('Set the app theme to System to capture both themes.')
    await mkdir(resolve(here, 'captures'), { recursive: true })
    const structureResponse = await fetch(`${api}/api/repositories/${structureRepoId}`)
    if (!structureResponse.ok) throw new Error(`Structure repository: HTTP ${structureResponse.status}`)
    const structureRepo = await structureResponse.json()
    if (structureRepo.status !== 'ready' || !/KhaledSaeed18\/repo-scout(?:\.git)?$/i.test(structureRepo.gitRemote || '')) {
      throw new Error('Scan Repo Scout itself for the architecture panel, or update the template captions.')
    }
    const describe = repository => ({ repository: repository.name, remote: repository.gitRemote,
      commit: repository.headCommit, files: repository.fileCount, commits: repository.commitCount,
      contributors: repository.contributorCount })
    const evidence = { capturedAt: new Date().toISOString(),
      applicationCommit: execFileSync('git', ['rev-parse', 'HEAD'], { cwd: root, encoding: 'utf8' }).trim(),
      repositories: [describe(repo), describe(structureRepo)], views: [] }

    for (const view of [
      { name: 'overview', path: '/', theme: 'dark', width: 1440, height: 1080, repo },
      { name: 'activity', path: '/activity', theme: 'light', width: 1280, height: 900, cropHeight: 680, repo },
      { name: 'architecture', path: '/architecture', theme: 'dark', width: 1280, height: 1000, cropGraph: true, repo: structureRepo },
      { name: 'knowledge', path: '/knowledge', title: 'Knowledge', theme: 'light', width: 1440, height: 1080, rows: 3, repo },
      { name: 'hotspots', path: '/hotspots', title: 'Hotspots', theme: 'light', width: 1440, height: 1080, rows: 5, cropSection: true, repo },
      { name: 'coupling', path: '/coupling', title: 'Change coupling', theme: 'dark', width: 1440, height: 1080, rows: 4, cropSection: true, repo },
    ]) {
      const context = await browser.newContext({ viewport: { width: view.width, height: view.height },
        deviceScaleFactor: 2, colorScheme: view.theme, reducedMotion: 'reduce' })
      const page = await context.newPage()
      const errors = []
      page.on('pageerror', error => errors.push(error.message))
      page.on('response', response => {
        if (response.url().includes('/api/') && response.status() >= 400) {
          errors.push(`${response.status()} ${response.url()}`)
        }
      })
      await page.goto(`${origin}${view.path}?repo=${view.repo.id}`)
      await page.getByRole('heading', { name: view.title || (view.name === 'overview' ? view.repo.name :
        view.name === 'activity' ? 'Activity' : 'Architecture'), exact: true }).waitFor()
      if (view.name === 'overview') {
        const hotspots = page.locator('main section').filter({ has: page.getByRole('heading', { name: 'Hotspots', exact: true }) })
        await hotspots.locator('li').first().waitFor()
        for (const label of ['Knowledge', 'Hotspots', 'Change coupling', 'Portfolio']) {
          await page.getByRole('link', { name: label, exact: true }).waitFor()
        }
      } else if (view.name === 'activity') {
        await page.getByRole('img', { name: /commits between/ }).waitFor()
      } else if (view.name === 'architecture') {
        await page.locator('.react-flow__node').first().waitFor()
        const nodes = page.locator('.react-flow__node')
        const labels = await nodes.allTextContents()
        const index = labels.findIndex(label => label === 'internal/analysis')
        if (index < 0) throw new Error('The expected analysis package is missing from the graph.')
        await nodes.nth(index).click()
        for (let i = 0; i < 3; i++) await page.getByRole('button', { name: 'zoom in', exact: true }).click()
        const canvas = await page.locator('.react-flow__pane').boundingBox()
        await page.mouse.move(canvas.x + canvas.width / 2, canvas.y + canvas.height - 60)
        await page.mouse.down()
        await page.mouse.move(canvas.x + canvas.width / 2 - 170, canvas.y + canvas.height - 60, { steps: 10 })
        await page.mouse.up()
        // Fit-view transitions must finish before taking the photograph.
        await page.waitForTimeout(600)
      } else {
        await page.locator('main section').first().locator('tbody tr').nth(view.rows - 1).waitFor()
      }
      await page.waitForLoadState('networkidle')
      await page.evaluate(() => document.fonts.ready)
      await page.waitForFunction(theme => document.documentElement.classList.contains('dark') ===
        (theme === 'dark'), view.theme)
      if (errors.length) throw new Error(errors.join('\n'))
      const bounds = await page.locator('main').boundingBox()
      let clip = view.cropHeight ? { x: bounds.x, y: bounds.y, width: bounds.width, height: view.cropHeight } : undefined
      if (view.cropGraph) {
        const graph = await page.locator('main section').first().boundingBox()
        clip = { x: bounds.x, y: bounds.y, width: bounds.width, height: graph.y + graph.height + 16 - bounds.y }
      }
      if (view.rows) {
        const section = page.locator('main section').first()
        const sectionBounds = await section.boundingBox()
        const row = await section.locator('tbody tr').nth(view.rows - 1).boundingBox()
        const top = view.cropSection ? sectionBounds.y - 12 : bounds.y
        clip = { x: bounds.x, y: top, width: bounds.width, height: row.y + row.height - top }
      }
      await page.screenshot({ path: resolve(here, `captures/${view.name}.png`), animations: 'disabled',
        ...(clip ? { clip } : {}) })
      evidence.views.push({ route: view.path, repository: view.repo.name, theme: view.theme,
        viewport: { width: view.width, height: view.height },
        ...(view.cropHeight ? { cropHeight: view.cropHeight } : {}),
        ...(view.rows ? { rows: view.rows } : {}),
        ...(clip ? { crop: clip } : {}),
        ...(view.name === 'architecture' ? { selectedFolder: 'internal/analysis', zoomInSteps: 3, panLeft: 170 } : {}) })
      console.log(`Captured ${view.name}: real ${view.repo.name} data, ${view.theme} theme`)
      await context.close()
    }
    await writeFile(resolve(here, 'capture.json'), JSON.stringify(evidence, null, 2) + '\n')
  }

  // Reuse the app's source tokens so the showcase cannot drift to a separate palette.
  const css = await readFile(resolve(root, 'frontend/src/index.css'), 'utf8')
  const light = css.match(/:root\s*\{[^}]+\}/)?.[0]
  const dark = css.match(/\.dark\s*\{[^}]+\}/)?.[0]
  if (!light || !dark) throw new Error('Could not find the app palette.')
  const html = (await readFile(resolve(here, 'showcase.html'), 'utf8'))
    .replace('</head>', `<style>${light}\n${dark}</style></head>`)
  const types = { '.html': 'text/html', '.png': 'image/png', '.svg': 'image/svg+xml',
    '.woff2': 'font/woff2' }
  const server = createServer(async (request, response) => {
    try {
      const path = decodeURIComponent(new URL(request.url, 'http://localhost').pathname)
      const allowed = path.startsWith('/docs/showcase/') || path.startsWith('/brand/') ||
        path.startsWith('/frontend/node_modules/@fontsource/')
      const file = resolve(root, `.${path}`)
      if (!allowed || relative(root, file).split(sep).includes('..')) {
        response.writeHead(404).end()
        return
      }
      const content = file === resolve(here, 'showcase.html') ? html : await readFile(file)
      response.writeHead(200, { 'Content-Type': types[extname(file)] || 'application/octet-stream' }).end(content)
    } catch {
      response.writeHead(404).end()
    }
  })
  await new Promise(resolve => server.listen(0, '127.0.0.1', resolve))
  try {
    const page = await browser.newPage({ viewport: { width: 1600, height: 1440 }, deviceScaleFactor: 2 })
    for (const view of ['overview', 'history', 'structure', 'insights']) {
      await page.goto(`http://127.0.0.1:${server.address().port}/docs/showcase/showcase.html?view=${view}`)
      await page.evaluate(() => document.fonts.ready)
      await page.waitForFunction(() => Array.from(document.images).every(image => image.complete && image.naturalWidth > 0))
      const plate = page.locator(`#${view}`)
      const collisions = await plate.evaluate(plate => {
        const footer = plate.querySelector('.footer').getBoundingClientRect()
        return Array.from(plate.querySelectorAll('.frame')).some(frame => frame.getBoundingClientRect().bottom > footer.top)
      })
      if (collisions) throw new Error(`${view}: screenshot overlaps footer; adjust plate height.`)
      await plate.screenshot({ path: resolve(here, `${view}.png`), animations: 'disabled' })
      console.log(`Rendered ${view}.png`)
    }
    await page.close()
  } finally {
    await new Promise(resolve => server.close(resolve))
  }
} finally {
  await browser.close()
}
