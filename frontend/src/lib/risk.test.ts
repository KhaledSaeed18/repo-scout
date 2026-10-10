import { describe, expect, it } from 'vitest'
import { describeBusFactor, folderLabel, hotspotWindows, linkLabel, partnerOf } from './risk'

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
