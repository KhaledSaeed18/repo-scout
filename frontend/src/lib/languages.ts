import type { Metrics } from './types'

const palette = [
  'var(--chart-1)',
  'var(--chart-2)',
  'var(--chart-3)',
  'var(--chart-4)',
  'var(--chart-5)',
  'var(--chart-6)',
  'var(--chart-7)',
]
export const otherColor = 'var(--chart-8)'

export interface LanguageShare {
  name: string
  files: number
  code: number
  share: number
  color: string
}

/**
 * Ranks languages by lines of code. The largest get distinct colors; the
 * long tail folds into one "Other" entry so the legend stays readable.
 * Files without a recognized language have no code lines and are skipped.
 */
export function rankLanguages(languages: Metrics['languages'], named = palette.length): LanguageShare[] {
  const rows = Object.entries(languages)
    .filter(([name, v]) => name !== '' && v.code > 0)
    .sort((a, b) => b[1].code - a[1].code || a[0].localeCompare(b[0]))
  const total = rows.reduce((sum, [, v]) => sum + v.code, 0)
  if (total === 0) return []

  const keep = rows.length > named ? named - 1 : rows.length
  const out: LanguageShare[] = rows.slice(0, keep).map(([name, v], i) => ({
    name,
    files: v.files,
    code: v.code,
    share: v.code / total,
    color: palette[i],
  }))
  const rest = rows.slice(keep)
  if (rest.length) {
    const code = rest.reduce((sum, [, v]) => sum + v.code, 0)
    const files = rest.reduce((sum, [, v]) => sum + v.files, 0)
    out.push({ name: 'Other', files, code, share: code / total, color: otherColor })
  }
  return out
}

/** Color for a language within a ranking, falling back to the "Other" tone. */
export function colorOf(ranking: LanguageShare[], language: string): string {
  return ranking.find((l) => l.name === language)?.color ?? otherColor
}
