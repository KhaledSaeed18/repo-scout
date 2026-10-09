import { formatNumber, formatPercent } from '@/lib/format'
import type { LanguageShare } from '@/lib/languages'

/**
 * The repository's composition as one banded strip, like strata in a core
 * sample, with a legend that doubles as the data table.
 */
export default function LanguageStrata({ languages }: { languages: LanguageShare[] }) {
  return (
    <div className="flex flex-col gap-4">
      <div
        role="img"
        aria-label={languages.map((l) => `${l.name} ${formatPercent(l.share)}`).join(', ')}
        className="flex h-9 w-full overflow-hidden rounded-sm bg-muted"
      >
        {languages.map((l) => (
          <div
            key={l.name}
            title={`${l.name}: ${formatPercent(l.share)}`}
            className="h-full border-r-2 border-background last:border-r-0"
            style={{ width: `${l.share * 100}%`, background: l.color }}
          />
        ))}
      </div>
      <ul className="grid grid-cols-[repeat(auto-fill,minmax(11rem,1fr))] gap-x-6 gap-y-2 text-sm">
        {languages.map((l) => (
          <li key={l.name} className="flex items-baseline gap-2">
            <span className="size-2.5 shrink-0 translate-y-px rounded-[2px]" style={{ background: l.color }} />
            <span className="font-medium">{l.name}</span>
            <span className="tabular-nums text-muted-foreground">{formatPercent(l.share)}</span>
            <span className="ml-auto tabular-nums text-xs text-muted-foreground" title="Lines of code">
              {formatNumber(l.code)}
            </span>
          </li>
        ))}
      </ul>
    </div>
  )
}
