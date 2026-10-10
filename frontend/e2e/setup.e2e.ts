import { execFileSync } from 'node:child_process'
import { mkdirSync, mkdtempSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { dirname, join } from 'node:path'
import { expect, test as setup } from '@playwright/test'

const duplicated = `package dup

func build(items []string) []string {
	out := []string{}
	for _, item := range items {
		if item == "" {
			continue
		}
		out = append(out, item)
	}
	return out
}
`

/**
 * Builds a small Git repository with two authors, an import, a duplicated
 * block and a file edited several times, then scans it through the API.
 */
setup('scan a fixture repository', async ({ request }) => {
  const root = mkdtempSync(join(tmpdir(), 'repo-scout-fixture-'))
  const write = (rel: string, content: string) => {
    mkdirSync(dirname(join(root, rel)), { recursive: true })
    writeFileSync(join(root, rel), content)
  }
  const commit = (who: string, date: string, message: string) => {
    const env = {
      ...process.env,
      GIT_AUTHOR_NAME: who,
      GIT_AUTHOR_EMAIL: `${who}@example.com`,
      GIT_COMMITTER_NAME: who,
      GIT_COMMITTER_EMAIL: `${who}@example.com`,
      GIT_AUTHOR_DATE: date,
      GIT_COMMITTER_DATE: date,
    }
    execFileSync('git', ['add', '.'], { cwd: root, env })
    execFileSync('git', ['commit', '-qm', message], { cwd: root, env })
  }

  execFileSync('git', ['init', '-q', '-b', 'main'], { cwd: root })
  write('go.mod', 'module example.com/fixture\n\ngo 1.22\n')
  write('main.go', 'package main\n\nimport "example.com/fixture/util"\n\nfunc main() { util.Help() }\n')
  write('util/util.go', 'package util\n\n// Help prints a greeting.\nfunc Help() {\n\tprintln("hello")\n}\n')
  write('dup/a.go', duplicated)
  write('dup/b.go', duplicated.replace('build', 'collect'))
  write('README.md', '# Fixture\n')
  commit('ana', '2024-01-02T10:00:00', 'initial layout')
  for (const [i, date] of ['2024-02-01T09:00:00', '2024-03-05T15:00:00', '2024-04-10T11:00:00'].entries()) {
    write('util/util.go', `package util\n\n// Help prints a greeting.\nfunc Help() {\n\tif ${i} > 0 {\n\t\tprintln("hello ${i}")\n\t}\n}\n`)
    write('main.go', `package main\n\nimport "example.com/fixture/util"\n\n// v${i}\nfunc main() { util.Help() }\n`)
    commit('bo', date, `tune help ${i}`)
  }

  const res = await request.post('/api/repositories', { data: { path: root } })
  expect(res.status()).toBe(202)
  await expect
    .poll(async () => (await (await request.get('/api/repositories')).json()).repositories[0]?.status, {
      timeout: 30_000,
    })
    .toBe('ready')
})
