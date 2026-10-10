import { describe, expect, it } from 'vitest'
import { describeBusFactor, folderLabel, hotspotWindows } from './risk'

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
