import { plural } from '@/lib/format'

const days = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday']
const maxDot = 18

/** Commits by weekday and hour, drawn as dots whose area tracks the count. */
export default function PunchCard({ hourly }: { hourly: number[][] }) {
  const max = Math.max(1, ...hourly.flat())
  return (
    <div className="overflow-x-auto pb-1">
      <table className="border-separate border-spacing-0 text-[0.6875rem] text-muted-foreground">
        <caption className="sr-only">Commits by day of week and hour</caption>
        <thead>
          <tr>
            <th />
            {Array.from({ length: 24 }, (_, h) => (
              <th key={h} scope="col" className="h-5 w-[26px] text-center font-normal">
                {h % 3 === 0 ? String(h).padStart(2, '0') : ''}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {hourly.map((row, d) => (
            <tr key={d}>
              <th scope="row" className="pr-3 text-left font-normal">
                {days[d].slice(0, 3)}
              </th>
              {row.map((count, h) => {
                const size = count ? Math.max(4, Math.sqrt(count / max) * maxDot) : 0
                return (
                  <td
                    key={h}
                    title={`${plural(count, 'commit')} on ${days[d]}s at ${String(h).padStart(2, '0')}:00`}
                    className="h-[26px] w-[26px] border-t border-border/60 text-center align-middle"
                  >
                    {count > 0 ? (
                      <span
                        className="inline-block rounded-full bg-contour-3"
                        style={{ width: size, height: size }}
                      />
                    ) : (
                      <span className="inline-block size-[3px] rounded-full bg-border" />
                    )}
                    <span className="sr-only">{count}</span>
                  </td>
                )
              })}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
