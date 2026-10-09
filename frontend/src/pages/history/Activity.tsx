import { useState } from 'react'
import { Button, Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui'
import CalendarHeatmap from '@/components/CalendarHeatmap'
import PunchCard from '@/components/PunchCard'
import RequireRepo from '@/components/RequireRepo'
import { PageHeader, Section } from '@/components/layout'
import { QueryView } from '@/components/states'
import { useContributors, useHeatmap } from '@/lib/api'
import { formatDate, formatNumber, plural } from '@/lib/format'
import type { Heatmap, Repository, Streak, StreaksResult } from '@/lib/types'

const latest = 'latest'

function Calendar({ heatmap }: { heatmap: Heatmap }) {
  const firstYear = Number(heatmap.start.slice(0, 4))
  const lastYear = Number(heatmap.end.slice(0, 4))
  const years = Array.from({ length: lastYear - firstYear + 1 }, (_, i) => String(lastYear - i))
  const [range, setRange] = useState(latest)
  const end = range === latest ? heatmap.end : `${range}-12-31`

  return (
    <Section
      title="Commit calendar"
      actions={
        years.length > 1 && (
          <div role="group" aria-label="Calendar range" className="flex flex-wrap gap-1">
            {[latest, ...years].map((y) => (
              <Button
                key={y}
                size="sm"
                variant={range === y ? 'secondary' : 'ghost'}
                aria-pressed={range === y}
                onClick={() => setRange(y)}
              >
                {y === latest ? 'Last 12 months' : y}
              </Button>
            ))}
          </div>
        )
      }
    >
      <CalendarHeatmap daily={heatmap.daily} end={end} />
    </Section>
  )
}

function span(s: Streak) {
  if (!s.days) return '—'
  const range = s.start === s.end ? formatDate(`${s.start}T12:00:00Z`) : `${formatDate(`${s.start}T12:00:00Z`)} to ${formatDate(`${s.end}T12:00:00Z`)}`
  return (
    <>
      <span className="font-medium">{plural(s.days, 'day')}</span>
      <span className="ml-2 text-muted-foreground">{range}</span>
    </>
  )
}

function Streaks({ repoId, streaks }: { repoId: number; streaks: StreaksResult[] }) {
  const contributors = useContributors(repoId)
  const nameOf = (email: string) => contributors.data?.contributors.find((c) => c.email === email)?.name || email
  return (
    <Section title="Streaks" description="Runs of consecutive days with at least one commit. A streak is current if it reaches today or yesterday.">
      {streaks.length === 0 ? (
        <p className="text-sm text-muted-foreground">No commits yet.</p>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Author</TableHead>
              <TableHead className="text-right">Commits</TableHead>
              <TableHead className="text-right">Active days</TableHead>
              <TableHead>Longest streak</TableHead>
              <TableHead>Current streak</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {streaks.map((s) => (
              <TableRow key={s.email}>
                <TableCell className="font-medium" title={s.email}>
                  {nameOf(s.email)}
                </TableCell>
                <TableCell className="text-right">{formatNumber(s.totalCommits)}</TableCell>
                <TableCell className="text-right">{formatNumber(s.activeDays)}</TableCell>
                <TableCell>{span(s.longest)}</TableCell>
                <TableCell>{span(s.current)}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </Section>
  )
}

function ActivityPage({ repo }: { repo: Repository }) {
  const heatmap = useHeatmap(repo.id)
  return (
    <>
      <PageHeader
        title="Activity"
        description={`When work happens in ${repo.name}. Days and hours use each author's own clock.`}
      />
      <QueryView query={heatmap} label="activity">
        {({ heatmap: h, streaks }) =>
          h.total === 0 ? (
            <p className="text-sm text-muted-foreground">This repository has no commits yet.</p>
          ) : (
            <>
              <Calendar heatmap={h} />
              <Section title="Hours of the week" description="Each dot's area is the number of commits started in that hour.">
                <PunchCard hourly={h.hourly} />
              </Section>
              <Streaks repoId={repo.id} streaks={streaks} />
            </>
          )
        }
      </QueryView>
    </>
  )
}

export default function Activity() {
  return <RequireRepo>{(repo) => <ActivityPage repo={repo} />}</RequireRepo>
}
