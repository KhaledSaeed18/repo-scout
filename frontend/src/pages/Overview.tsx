import { RefreshCw } from 'lucide-react'
import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { Button } from '@/components/ui'
import CalendarHeatmap from '@/components/CalendarHeatmap'
import LanguageStrata from '@/components/LanguageStrata'
import Meter from '@/components/Meter'
import RequireRepo from '@/components/RequireRepo'
import { PageHeader, Section } from '@/components/layout'
import { QueryView } from '@/components/states'
import { useContributors, useCreateRepo, useHeatmap, useHotspots, useMetrics } from '@/lib/api'
import { formatBytes, formatCompact, formatNumber, formatRelative, formatRemote, plural, shortHash } from '@/lib/format'
import { rankLanguages } from '@/lib/languages'
import type { Repository } from '@/lib/types'

function Fact({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="min-w-0">
      <dt className="text-xs text-muted-foreground">{label}</dt>
      <dd className="truncate">{children}</dd>
    </div>
  )
}

function Figure({ value, label, to }: { value: number; label: string; to: string }) {
  return (
    <Link to={to} className="group flex flex-col gap-0.5 px-5 py-3 first:pl-0 hover:text-primary">
      <span className="font-heading text-3xl leading-none font-semibold tabular-nums" title={formatNumber(value)}>
        {formatCompact(value)}
      </span>
      <span className="text-sm text-muted-foreground group-hover:text-primary">{label}</span>
    </Link>
  )
}

function Header({ repo }: { repo: Repository }) {
  const rescan = useCreateRepo()
  const remote = formatRemote(repo.gitRemote)
  return (
    <>
      <PageHeader
        title={repo.name}
        description={<span className="font-mono text-sm break-all">{repo.path}</span>}
        actions={
          <Button
            variant="outline"
            disabled={repo.status === 'scanning' || rescan.isPending}
            onClick={() => rescan.mutate(repo.path)}
          >
            <RefreshCw className={repo.status === 'scanning' ? 'animate-spin' : undefined} />
            {repo.status === 'scanning' ? 'Scanning…' : 'Scan again'}
          </Button>
        }
      />
      <dl className="-mt-4 mb-8 grid grid-cols-2 gap-x-8 gap-y-3 text-sm sm:grid-cols-3 lg:grid-cols-5">
        <Fact label="Branch">{repo.defaultBranch || 'Detached'}</Fact>
        <Fact label="Head commit">
          <span className="font-mono">{repo.headCommit ? shortHash(repo.headCommit) : 'None'}</span>
        </Fact>
        <Fact label="Remote">{remote || 'None'}</Fact>
        <Fact label="Last scan">{repo.lastScannedAt ? formatRelative(repo.lastScannedAt) : 'Never'}</Fact>
        <Fact label="Size on disk">{formatBytes(repo.totalSize)}</Fact>
      </dl>
      <div className="mb-2 flex flex-wrap divide-x border-y">
        <Figure value={repo.totalCode} label="lines of code" to="/metrics" />
        <Figure value={repo.fileCount} label="files" to="/files" />
        <Figure value={repo.commitCount} label="commits" to="/commits" />
        <Figure value={repo.contributorCount} label={repo.contributorCount === 1 ? 'contributor' : 'contributors'} to="/contributors" />
        <Figure value={repo.dependencyCount} label="dependencies" to="/dependencies" />
        <Figure value={repo.dupGroupCount} label="duplicate blocks" to="/duplicates" />
      </div>
    </>
  )
}

function Languages({ repoId }: { repoId: number }) {
  const metrics = useMetrics(repoId, 6)
  return (
    <Section title="Languages" description="Share of lines of code, excluding comments and blank lines.">
      <QueryView query={metrics} label="languages">
        {(m) => {
          const ranked = rankLanguages(m.languages)
          return ranked.length ? (
            <LanguageStrata languages={ranked} />
          ) : (
            <p className="text-sm text-muted-foreground">No source code was recognized in this repository.</p>
          )
        }}
      </QueryView>
    </Section>
  )
}

function Activity({ repoId }: { repoId: number }) {
  const heatmap = useHeatmap(repoId)
  return (
    <Section
      title="Commit activity"
      actions={
        <Link to="/activity" className="text-sm text-primary hover:underline">
          Open activity
        </Link>
      }
    >
      <QueryView query={heatmap} label="commit activity">
        {({ heatmap: h }) =>
          h.total ? (
            <CalendarHeatmap daily={h.daily} end={h.end} />
          ) : (
            <p className="text-sm text-muted-foreground">This repository has no commits yet.</p>
          )
        }
      </QueryView>
    </Section>
  )
}

function Hotspots({ repoId }: { repoId: number }) {
  const q = useHotspots(repoId, 12, 6)
  return (
    <Section
      title="Hotspots"
      description="Complex files changed most in the last year of history."
      actions={
        <Link to="/hotspots" className="text-sm text-primary hover:underline">
          All hotspots
        </Link>
      }
    >
      <QueryView query={q} label="hotspots">
        {({ hotspots, until }) => {
          if (!until) return <p className="text-sm text-muted-foreground">Hotspots need Git history.</p>
          if (!hotspots.length) return <p className="text-sm text-muted-foreground">No analyzed file changed in the last year of history.</p>
          return (
            <ol className="flex flex-col gap-3">
              {hotspots.map((h) => (
                <li key={h.path} className="flex flex-col gap-1">
                  <div className="flex items-baseline justify-between gap-3 text-sm">
                    <Link
                      to={`/files?file=${encodeURIComponent(h.path)}`}
                      className="truncate font-mono text-[0.8125rem] hover:text-primary hover:underline"
                      title={h.path}
                    >
                      {h.path}
                    </Link>
                    <span className="shrink-0 tabular-nums text-muted-foreground">{plural(h.revisions, 'change')}</span>
                  </div>
                  <Meter value={h.score} max={1} />
                </li>
              ))}
            </ol>
          )
        }}
      </QueryView>
    </Section>
  )
}

function Contributors({ repoId }: { repoId: number }) {
  const contributors = useContributors(repoId)
  return (
    <Section
      title="Contributors"
      description="By number of commits."
      actions={
        <Link to="/contributors" className="text-sm text-primary hover:underline">
          Everyone
        </Link>
      }
    >
      <QueryView query={contributors} label="contributors">
        {({ contributors: people }) => {
          if (!people.length) return <p className="text-sm text-muted-foreground">No commits yet.</p>
          const top = people.slice(0, 6)
          const max = top[0].commits
          return (
            <ol className="flex flex-col gap-3">
              {top.map((c) => (
                <li key={c.email || c.name} className="flex flex-col gap-1">
                  <div className="flex items-baseline justify-between gap-3 text-sm">
                    <span className="truncate font-medium" title={c.email}>
                      {c.name || c.email}
                    </span>
                    <span className="shrink-0 tabular-nums text-muted-foreground">{plural(c.commits, 'commit')}</span>
                  </div>
                  <Meter value={c.commits} max={max} />
                </li>
              ))}
            </ol>
          )
        }}
      </QueryView>
    </Section>
  )
}

export default function Overview() {
  return (
    <RequireRepo>
      {(repo) => (
        <>
          <Header repo={repo} />
          <Languages repoId={repo.id} />
          <Activity repoId={repo.id} />
          <div className="mt-10 grid grid-cols-1 gap-x-10 gap-y-10 lg:grid-cols-2 [&>section]:mt-0">
            <Hotspots repoId={repo.id} />
            <Contributors repoId={repo.id} />
          </div>
        </>
      )}
    </RequireRepo>
  )
}
