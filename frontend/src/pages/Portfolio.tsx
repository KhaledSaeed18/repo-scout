import { FolderGit2 } from 'lucide-react'
import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { buttonVariants, Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui'
import Meter from '@/components/Meter'
import OptionSelect from '@/components/OptionSelect'
import { PageHeader, Section } from '@/components/layout'
import { Empty, QueryView } from '@/components/states'
import { usePortfolio } from '@/lib/api'
import { formatCompact, formatNumber, formatPercent, plural } from '@/lib/format'
import { useRepoContext } from '@/lib/repo-context'
import { portfolioSorts, signed, sortPortfolio } from '@/lib/risk'
import type { Portfolio, PortfolioEntry } from '@/lib/types'

function Figure({ value, label }: { value: string; label: string }) {
  return (
    <div className="flex flex-col gap-0.5 px-5 py-3 first:pl-0">
      <span className="font-heading text-3xl leading-none font-semibold tabular-nums">{value}</span>
      <span className="text-sm text-muted-foreground">{label}</span>
    </div>
  )
}

function Totals({ p }: { p: Portfolio }) {
  const sum = (f: (e: PortfolioEntry) => number) => p.entries.reduce((s, e) => s + f(e), 0)
  return (
    <div className="mb-2 flex flex-wrap divide-x border-y">
      <Figure value={formatNumber(p.entries.length)} label={p.entries.length === 1 ? 'repository' : 'repositories'} />
      <Figure value={formatCompact(sum((e) => e.repository.totalCode))} label="lines of code" />
      <Figure value={formatCompact(sum((e) => e.repository.commitCount))} label="commits" />
      <Figure value={formatNumber(p.entries.filter((e) => e.busFactor === 1).length)} label="with a bus factor of 1" />
      <Figure value={formatNumber(sum((e) => e.cycles))} label="circular dependencies" />
    </div>
  )
}

function Change({ n }: { n: number | null }) {
  const text = signed(n)
  return text ? <div className="text-xs text-muted-foreground">{text} since the last scan</div> : null
}

function Entries({ entries }: { entries: PortfolioEntry[] }) {
  const { setRepoId } = useRepoContext()
  const navigate = useNavigate()
  const open = (id: number, to: string) => {
    setRepoId(id)
    navigate(to)
  }
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Repository</TableHead>
          <TableHead className="text-right">Lines of code</TableHead>
          <TableHead className="text-right">Complexity</TableHead>
          <TableHead className="text-right">Bus factor</TableHead>
          <TableHead className="w-36">Inactive authors</TableHead>
          <TableHead className="text-right">Cycles</TableHead>
          <TableHead className="text-right">Hidden dependencies</TableHead>
          <TableHead>Top hotspot</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {entries.map((e) => {
          const r = e.repository
          return (
            <TableRow key={r.id}>
              <TableCell className="max-w-64">
                <button
                  className="cursor-pointer text-left font-medium text-primary hover:underline"
                  onClick={() => open(r.id, '/')}
                >
                  {r.name}
                </button>
                <div className="truncate font-mono text-xs text-muted-foreground" title={r.path}>
                  {r.path}
                </div>
              </TableCell>
              <TableCell className="text-right">
                {formatNumber(r.totalCode)}
                <Change n={e.linesChange} />
              </TableCell>
              <TableCell className="text-right">
                {formatNumber(e.complexity)}
                <Change n={e.complexityChange} />
              </TableCell>
              <TableCell className="text-right">
                <button className="cursor-pointer hover:text-primary hover:underline" onClick={() => open(r.id, '/knowledge')}>
                  {formatNumber(e.busFactor)}
                </button>
              </TableCell>
              <TableCell>
                <div className="flex items-center gap-2">
                  <Meter value={e.inactiveShare} max={1} />
                  <span className="w-12 shrink-0 text-right text-xs text-muted-foreground">{formatPercent(e.inactiveShare)}</span>
                </div>
              </TableCell>
              <TableCell className="text-right">
                <button className="cursor-pointer hover:text-primary hover:underline" onClick={() => open(r.id, '/architecture')}>
                  {formatNumber(e.cycles)}
                </button>
              </TableCell>
              <TableCell className="text-right">
                <button className="cursor-pointer hover:text-primary hover:underline" onClick={() => open(r.id, '/coupling')}>
                  {e.hiddenDependencies >= 500 ? '500+' : formatNumber(e.hiddenDependencies)}
                </button>
              </TableCell>
              <TableCell className="max-w-56">
                {e.topHotspot ? (
                  <button
                    className="block max-w-full cursor-pointer truncate text-left font-mono text-[0.8125rem] text-primary hover:underline"
                    title={e.topHotspot}
                    onClick={() => open(r.id, `/files?file=${encodeURIComponent(e.topHotspot)}`)}
                  >
                    {e.topHotspot}
                  </button>
                ) : (
                  <span className="text-muted-foreground">None</span>
                )}
              </TableCell>
            </TableRow>
          )
        })}
      </TableBody>
    </Table>
  )
}

export default function PortfolioPage() {
  const q = usePortfolio()
  const [sort, setSort] = useState('name')
  return (
    <>
      <PageHeader
        title="Portfolio"
        description="Every scanned repository side by side: how big it is, which way it moved since its last scan, and where knowledge and structure are at risk."
        actions={<OptionSelect label="Sort by" value={sort} options={portfolioSorts} onChange={setSort} className="w-52" />}
      />
      <QueryView query={q} label="portfolio">
        {(p) =>
          p.entries.length === 0 ? (
            <Empty icon={FolderGit2} title="No finished scans yet" className="mt-4">
              <p className="mb-3">
                {p.unscanned ? `${plural(p.unscanned, 'repository is', 'repositories are')} still waiting for a first scan.` : 'Add a repository to start.'}
              </p>
              <Link to="/repositories" className={buttonVariants({ variant: 'outline' })}>
                Go to repositories
              </Link>
            </Empty>
          ) : (
            <>
              <Totals p={p} />
              <Section
                title="Repositories"
                description={
                  p.unscanned ? `${plural(p.unscanned, 'more repository is', 'more repositories are')} waiting for a first scan.` : undefined
                }
              >
                <Entries entries={sortPortfolio(p.entries, sort)} />
              </Section>
            </>
          )
        }
      </QueryView>
    </>
  )
}
