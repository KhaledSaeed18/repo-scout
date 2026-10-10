import { Flame } from 'lucide-react'
import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui'
import Meter from '@/components/Meter'
import OptionSelect from '@/components/OptionSelect'
import RequireRepo from '@/components/RequireRepo'
import { PageHeader, Section } from '@/components/layout'
import { Empty, QueryView } from '@/components/states'
import { useHotspots } from '@/lib/api'
import { formatDate, formatNumber, formatRelative, plural } from '@/lib/format'
import { hotspotWindows } from '@/lib/risk'
import type { HotspotReport, Repository } from '@/lib/types'

function windowText(r: HotspotReport): string {
  if (!r.until) return ''
  const range = r.since
    ? `between ${formatDate(r.since)} and ${formatDate(r.until)}, the latest commit`
    : `across all history up to ${formatDate(r.until)}`
  return `${plural(r.files, 'analyzed file')} changed ${range}.`
}

function Ranking({ report }: { report: HotspotReport }) {
  if (!report.until) {
    return (
      <Empty icon={Flame} title="No history to rank by">
        Hotspots need Git history. Scan a folder that is a Git repository to see them.
      </Empty>
    )
  }
  if (!report.hotspots.length) {
    return (
      <Empty icon={Flame} title="No analyzed file changed in this window">
        Pick a longer window to include older changes. Repositories scanned before hotspots existed need a new scan.
      </Empty>
    )
  }
  return (
    <Section title="Ranked files" description={windowText(report)}>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>File</TableHead>
            <TableHead className="text-right">Changes</TableHead>
            <TableHead className="text-right">Complexity</TableHead>
            <TableHead className="text-right">Lines changed</TableHead>
            <TableHead className="text-right">Authors</TableHead>
            <TableHead>Last change</TableHead>
            <TableHead className="w-32">Score</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {report.hotspots.map((h) => (
            <TableRow key={h.path}>
              <TableCell className="max-w-96">
                <Link
                  to={`/files?file=${encodeURIComponent(h.path)}`}
                  className="block truncate font-mono text-[0.8125rem] text-primary hover:underline"
                  title={h.path}
                >
                  {h.path}
                </Link>
              </TableCell>
              <TableCell className="text-right">{formatNumber(h.revisions)}</TableCell>
              <TableCell className="text-right">{formatNumber(h.complexity)}</TableCell>
              <TableCell className="text-right">{formatNumber(h.churn)}</TableCell>
              <TableCell className="text-right">{formatNumber(h.authors)}</TableCell>
              <TableCell className="whitespace-nowrap text-muted-foreground">
                {h.lastCommitAt ? formatRelative(h.lastCommitAt) : 'Untracked'}
              </TableCell>
              <TableCell>
                <div className="flex items-center gap-2">
                  <Meter value={h.score} max={1} />
                  <span className="w-8 shrink-0 text-right text-xs text-muted-foreground">{Math.round(h.score * 100)}</span>
                </div>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </Section>
  )
}

function HotspotsPage({ repo }: { repo: Repository }) {
  const [months, setMonths] = useState('12')
  const q = useHotspots(repo.id, Number(months))
  return (
    <>
      <PageHeader
        title="Hotspots"
        description="Complex files that keep changing. Complex code nobody touches costs little; these files are where changes are slow and bugs tend to appear. Scores compare each file with the top one."
        actions={<OptionSelect label="Count changes from" value={months} options={hotspotWindows} onChange={setMonths} />}
      />
      <QueryView query={q} label="hotspots">
        {(report) => <Ranking report={report} />}
      </QueryView>
    </>
  )
}

export default function Hotspots() {
  return <RequireRepo>{(repo) => <HotspotsPage repo={repo} />}</RequireRepo>
}
