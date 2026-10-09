import { describe, expect, it } from 'vitest'
import { colorOf, otherColor, rankLanguages } from './languages'

const langs = {
  '': { files: 8, loc: 0, code: 0 },
  Go: { files: 47, loc: 8002, code: 6000 },
  TypeScript: { files: 37, loc: 4099, code: 3000 },
  YAML: { files: 3, loc: 4586, code: 1000 },
  CSS: { files: 1, loc: 155, code: 0 },
}

describe('rankLanguages', () => {
  it('orders by code lines, drops empty languages and computes shares', () => {
    const ranked = rankLanguages(langs)
    expect(ranked.map((l) => l.name)).toEqual(['Go', 'TypeScript', 'YAML'])
    expect(ranked[0].share).toBeCloseTo(0.6)
    expect(ranked[0].color).toBe('var(--chart-1)')
  })

  it('folds the tail into Other when there are more languages than colors', () => {
    const ranked = rankLanguages(langs, 2)
    expect(ranked.map((l) => l.name)).toEqual(['Go', 'Other'])
    expect(ranked[1]).toMatchObject({ code: 4000, files: 40, color: otherColor })
    expect(ranked.reduce((s, l) => s + l.share, 0)).toBeCloseTo(1)
  })

  it('returns nothing when no code was counted', () => {
    expect(rankLanguages({ '': { files: 2, loc: 0, code: 0 } })).toEqual([])
  })
})

it('colors unranked languages as Other', () => {
  const ranked = rankLanguages(langs)
  expect(colorOf(ranked, 'TypeScript')).toBe('var(--chart-2)')
  expect(colorOf(ranked, 'Haskell')).toBe(otherColor)
})
