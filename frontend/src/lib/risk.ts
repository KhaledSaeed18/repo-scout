import type { CouplingLink, PortfolioEntry } from './types'

/** Windows offered for counting changes, in months; 0 means all history. */
export const hotspotWindows = [
  { value: '3', label: 'Last 3 months' },
  { value: '6', label: 'Last 6 months' },
  { value: '12', label: 'Last 12 months' },
  { value: '24', label: 'Last 2 years' },
  { value: '0', label: 'All history' },
]

/** How long without a commit before an author counts as inactive. */
export const inactiveWindows = [
  { value: '3', label: '3 months' },
  { value: '6', label: '6 months' },
  { value: '12', label: '12 months' },
  { value: '24', label: '2 years' },
]

/** How deep folders are grouped. */
export const folderDepths = [
  { value: '1', label: 'Top folders' },
  { value: '2', label: 'Two levels' },
  { value: '3', label: 'Three levels' },
]

/** Files at the repository root group under an empty folder name. */
export function folderLabel(folder: string): string {
  return folder === '' ? 'Top level' : folder
}

/** Plain reading of a bus factor for the figure under it. */
export function describeBusFactor(n: number): string {
  if (n <= 0) return 'No files have a main author yet.'
  if (n === 1) return 'One person is the main author of most of the code.'
  if (n === 2) return 'Two people are the main authors of most of the code.'
  return `Most of the code is spread across ${n} main authors.`
}

/** Which coupled pairs to list. */
export const couplingFilters = [
  { value: 'all', label: 'All pairs' },
  { value: 'hidden', label: 'Hidden only' },
]

const linkLabels: Record<CouplingLink, string> = {
  '': 'Hidden',
  import: 'Import',
  package: 'Same package',
  test: 'Test',
  lockfile: 'Lockfile',
}

/** Short label for why a coupled pair changes together. */
export function linkLabel(link: CouplingLink): string {
  return linkLabels[link] ?? link
}

/** The partner of path in a coupled pair. */
export function partnerOf(pair: { fileA: string; fileB: string }, path: string): string {
  return pair.fileA === path ? pair.fileB : pair.fileA
}

/** Orders the portfolio offers. */
export const portfolioSorts = [
  { value: 'name', label: 'Name' },
  { value: 'size', label: 'Largest' },
  { value: 'busFactor', label: 'Lowest bus factor' },
  { value: 'hidden', label: 'Most hidden dependencies' },
  { value: 'growth', label: 'Fastest growing' },
]

/** Sorts portfolio entries without mutating them; name breaks ties. */
export function sortPortfolio(entries: PortfolioEntry[], by: string): PortfolioEntry[] {
  const key: Record<string, (e: PortfolioEntry) => number> = {
    size: (e) => -e.repository.totalCode,
    busFactor: (e) => e.busFactor,
    hidden: (e) => -e.hiddenDependencies,
    growth: (e) => -(e.linesChange ?? 0),
  }
  const k = key[by]
  return [...entries].sort(
    (a, b) => (k ? k(a) - k(b) : 0) || a.repository.name.localeCompare(b.repository.name),
  )
}

/** "+1,204", "−3" or "" when unknown or unchanged. */
export function signed(n: number | null): string {
  if (!n) return ''
  return `${n > 0 ? '+' : '−'}${Math.abs(n).toLocaleString('en-US')}`
}
