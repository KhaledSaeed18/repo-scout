const integer = new Intl.NumberFormat('en-US')
const compact = new Intl.NumberFormat('en-US', { notation: 'compact', maximumFractionDigits: 1 })
const shortDate = new Intl.DateTimeFormat('en-US', { month: 'short', day: 'numeric', year: 'numeric' })

/** 14916 -> "14,916" */
export function formatNumber(n: number): string {
  return integer.format(n)
}

/** 14916 -> "14.9K"; values under 10,000 keep full precision. */
export function formatCompact(n: number): string {
  return Math.abs(n) < 10_000 ? integer.format(n) : compact.format(n)
}

const byteUnits = ['B', 'KB', 'MB', 'GB', 'TB']

/** Binary-scaled size with one decimal: 1572036 -> "1.5 MB". */
export function formatBytes(n: number): string {
  let value = n
  let unit = 0
  while (value >= 1024 && unit < byteUnits.length - 1) {
    value /= 1024
    unit++
  }
  return unit === 0 ? `${value} B` : `${value.toFixed(1)} ${byteUnits[unit]}`
}

/** 0.4567 -> "45.7%"; tiny non-zero shares read as "<0.1%". */
export function formatPercent(share: number): string {
  if (share > 0 && share < 0.001) return '<0.1%'
  return `${(share * 100).toFixed(1)}%`
}

export function shortHash(hash: string): string {
  return hash.slice(0, 7)
}

const dayOnly = /^(\d{4})-(\d{2})-(\d{2})$/

/** "Aug 7, 2026". Date-only strings are calendar days, not UTC midnights. */
export function formatDate(value: string | Date): string {
  const day = typeof value === 'string' ? dayOnly.exec(value) : null
  return shortDate.format(day ? new Date(+day[1], +day[2] - 1, +day[3]) : new Date(value))
}

/** Human distance from now, falling back to a calendar date past 30 days. */
export function formatRelative(value: string | Date, now: Date = new Date()): string {
  const then = new Date(value)
  const seconds = Math.round((now.getTime() - then.getTime()) / 1000)
  if (seconds < 45) return 'just now'
  const minutes = Math.round(seconds / 60)
  if (minutes < 60) return `${minutes} min ago`
  const hours = Math.round(minutes / 60)
  if (hours < 24) return hours === 1 ? '1 hour ago' : `${hours} hours ago`
  const days = Math.round(hours / 24)
  if (days === 1) return 'yesterday'
  if (days <= 30) return `${days} days ago`
  return formatDate(then)
}

/** Pluralizes a unit noun: plural(1, 'file') -> "1 file". */
export function plural(n: number, singular: string, pluralForm = `${singular}s`): string {
  return `${formatNumber(n)} ${n === 1 ? singular : pluralForm}`
}

/** Normalizes HTTPS and SSH remotes to "host/owner/repo". */
export function formatRemote(remote: string): string {
  return remote
    .trim()
    .replace(/^[a-z+]+:\/\//, '')
    .replace(/^[^@/]+@/, '')
    .replace(/^([^/:]+):(?!\d+\/)/, '$1/')
    .replace(/\.git$/, '')
}
