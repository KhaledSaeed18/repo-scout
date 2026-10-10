import { describe, expect, it } from 'vitest'
import { describeBusFactor, folderLabel, hotspotWindows, linkLabel, partnerOf, signed, sortPortfolio } from './risk'
import type { PortfolioEntry } from './types'

describe('folderLabel', () => {
  it('names the repository root', () => {
    expect(folderLabel('')).toBe('Top level')
    expect(folderLabel('internal/api')).toBe('internal/api')
  })
})

describe('describeBusFactor', () => {
  it('reads each size in plain words', () => {
    expect(describeBusFactor(0)).toMatch(/no files/i)
    expect(describeBusFactor(1)).toMatch(/^One person/)
    expect(describeBusFactor(2)).toMatch(/^Two people/)
    expect(describeBusFactor(5)).toContain('5 main authors')
  })
})

describe('hotspotWindows', () => {
  it('ends with all history', () => {
    expect(hotspotWindows.at(-1)).toEqual({ value: '0', label: 'All history' })
  })
})

describe('partnerOf', () => {
  it('returns the other file of the pair', () => {
    const pair = { fileA: 'a.ts', fileB: 'b.ts' }
    expect(partnerOf(pair, 'a.ts')).toBe('b.ts')
    expect(partnerOf(pair, 'b.ts')).toBe('a.ts')
  })
})

describe('linkLabel', () => {
  it('names hidden pairs and each reason', () => {
    expect(linkLabel('')).toBe('Hidden')
    expect(linkLabel('package')).toBe('Same package')
  })
})

const entry = (name: string, totalCode: number, busFactor: number, linesChange: number | null): PortfolioEntry =>
  ({
    repository: { name, totalCode } as PortfolioEntry['repository'],
    complexity: 0,
    linesChange,
    complexityChange: null,
    busFactor,
    inactiveShare: 0,
    cycles: 0,
    hiddenDependencies: 0,
    topHotspot: '',
  })

describe('sortPortfolio', () => {
  const entries = [entry('b', 10, 2, 5), entry('a', 30, 1, null), entry('c', 20, 1, 50)]
  it('sorts by each key with name as the tie breaker', () => {
    expect(sortPortfolio(entries, 'name').map((e) => e.repository.name)).toEqual(['a', 'b', 'c'])
    expect(sortPortfolio(entries, 'size').map((e) => e.repository.name)).toEqual(['a', 'c', 'b'])
    expect(sortPortfolio(entries, 'busFactor').map((e) => e.repository.name)).toEqual(['a', 'c', 'b'])
    expect(sortPortfolio(entries, 'growth').map((e) => e.repository.name)).toEqual(['c', 'b', 'a'])
  })
  it('leaves the input untouched', () => {
    sortPortfolio(entries, 'size')
    expect(entries[0].repository.name).toBe('b')
  })
})

describe('signed', () => {
  it('shows a sign and nothing for no change', () => {
    expect(signed(1204)).toBe('+1,204')
    expect(signed(-3)).toBe('−3')
    expect(signed(0)).toBe('')
    expect(signed(null)).toBe('')
  })
})
