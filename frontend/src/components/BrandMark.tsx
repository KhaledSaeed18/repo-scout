import { cn } from '@/lib/utils'

/**
 * The Repo Scout mark at interface sizes (32px and below), using the
 * two-contour cut from brand/mark-small.svg. Rings follow the text color and
 * the summit uses the primary color, so it adapts to both themes.
 */
export default function BrandMark({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 64 64" className={cn('size-7 shrink-0', className)} aria-hidden>
      <g fill="none" stroke="currentColor" strokeWidth="5.2">
        <ellipse cx="31" cy="33" rx="26.95" ry="24.5" transform="rotate(-35 31 33)" />
        <ellipse cx="35.6" cy="28.4" rx="12.65" ry="11.5" transform="rotate(-35 35.6 28.4)" />
      </g>
      <circle cx="37.43" cy="26.57" r="5" fill="var(--primary)" />
    </svg>
  )
}
