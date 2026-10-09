import { buildCalendar, type Level } from '@/lib/calendar'
import { formatDate, plural } from '@/lib/format'
import type { DayActivity } from '@/lib/types'
import { cn } from '@/lib/utils'

const levelClass: Record<Level, string> = {
  0: 'bg-contour-0',
  1: 'bg-contour-1',
  2: 'bg-contour-2',
  3: 'bg-contour-3',
  4: 'bg-contour-4',
}

const dayLabels = ['', 'Mon', '', 'Wed', '', 'Fri', '']

/** Contribution calendar: one square per day, one column per week. */
export default function CalendarHeatmap({
  daily,
  end,
  weeks = 53,
}: {
  daily: DayActivity[]
  end: string
  weeks?: number
}) {
  const cal = buildCalendar(daily, end, weeks)
  const summary = `${plural(cal.total, 'commit')} between ${formatDate(cal.start)} and ${formatDate(cal.end)}`

  return (
    <figure className="flex flex-col gap-2">
      <div className="overflow-x-auto pb-1">
        <div className="inline-grid grid-cols-[auto_1fr] gap-x-2 text-[0.6875rem] text-muted-foreground">
          <span />
          <div className="relative h-4">
            {cal.months.map((m) => (
              <span key={`${m.label}-${m.week}`} className="absolute" style={{ left: m.week * 14 }}>
                {m.label}
              </span>
            ))}
          </div>
          <div className="grid grid-rows-7 gap-[3px]">
            {dayLabels.map((d, i) => (
              <span key={i} className="h-[11px] leading-[11px]">
                {d}
              </span>
            ))}
          </div>
          <div role="img" aria-label={summary} className="flex gap-[3px]">
            {cal.weeks.map((week, w) => (
              <div key={w} className="grid grid-rows-7 gap-[3px]">
                {week.map((cell, d) =>
                  cell ? (
                    <span
                      key={cell.date}
                      title={`${plural(cell.count, 'commit')} on ${formatDate(cell.date + 'T12:00:00Z')}`}
                      className={cn('size-[11px] rounded-[2px]', levelClass[cell.level])}
                    />
                  ) : (
                    <span key={`empty-${d}`} className="size-[11px]" />
                  ),
                )}
              </div>
            ))}
          </div>
        </div>
      </div>
      <figcaption className="flex flex-wrap items-center justify-between gap-2 text-xs text-muted-foreground">
        <span>{summary}</span>
        <span className="flex items-center gap-1">
          Fewer
          {([0, 1, 2, 3, 4] as Level[]).map((l) => (
            <span key={l} className={cn('size-[11px] rounded-[2px]', levelClass[l])} />
          ))}
          More
        </span>
      </figcaption>
    </figure>
  )
}
