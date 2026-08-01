import { useState } from 'react'
import {
  Bar,
  BarChart,
  Cell,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { Card, Empty, Spinner, Table, Tabs } from '../components/ui'
import { useCommits, useContributors, useHeatmap, useRepo } from '../lib/api'
import { RepoProvider, useRepoContext } from '../lib/repoctx'
import RepoSelector from '../components/RepoSelector'

function HeatmapGrid({ heatmap }: { heatmap: import('../lib/types').Heatmap }) {
  const max = Math.max(1, ...heatmap.daily.map((d) => d.count))
  const weeks: { date: string; count: number }[][] = []
  let week: { date: string; count: number }[] = []
  for (const d of heatmap.daily) {
    week.push(d)
    if (week.length === 7) {
      weeks.push(week)
      week = []
    }
  }
  if (week.length) weeks.push(week)
  const level = (count: number) => {
    const r = count / max
    if (count === 0) return 'bg-slate-800'
    if (r > 0.66) return 'bg-emerald-500'
    if (r > 0.33) return 'bg-emerald-600'
    return 'bg-emerald-800'
  }
  return (
    <Card>
      <h3 className="mb-3 text-sm font-medium text-slate-300">
        Commit activity ({heatmap.total} commits, {heatmap.start} → {heatmap.end})
      </h3>
      <div className="flex gap-1 overflow-x-auto">
        {weeks.map((w, i) => (
          <div key={i} className="flex flex-col gap-1">
            {w.map((d) => (
              <div
                key={d.date}
                title={`${d.date}: ${d.count}`}
                className={`size-3 rounded-sm ${level(d.count)}`}
              />
            ))}
          </div>
        ))}
      </div>
    </Card>
  )
}

function HourlyHeatmap({ hourly }: { hourly: number[][] }) {
  const data = hourly.flatMap((row, wd) =>
    row.map((count, hr) => ({ wd, hr, count })),
  )
  const max = Math.max(1, ...data.map((d) => d.count))
  return (
    <Card>
      <h3 className="mb-3 text-sm font-medium text-slate-300">Hour of week</h3>
      <ResponsiveContainer width="100%" height={180}>
        <BarChart data={data}>
          <XAxis dataKey="hr" hide />
          <YAxis hide />
          <Tooltip
            contentStyle={{ background: '#0f172a', border: '1px solid #334155' }}
            formatter={(value) => [Number(value), 'commits']}
            labelFormatter={(_, p) => {
              const d = p?.[0]?.payload as { wd: number; hr: number }
              return d ? `weekday ${d.wd} · ${d.hr}:00` : ''
            }}
          />
          <Bar dataKey="count" radius={[2, 2, 0, 0]}>
            {data.map((d, i) => (
              <Cell key={i} fill={d.count / max > 0.5 ? '#38bdf8' : '#0ea5e9'} />
            ))}
          </Bar>
        </BarChart>
      </ResponsiveContainer>
    </Card>
  )
}

function StreaksView({ streaks }: { streaks: import('../lib/types').StreaksResult[] }) {
  if (!streaks.length) return <Empty message="No streak data" />
  return (
    <Card>
      <h3 className="mb-3 text-sm font-medium text-slate-300">Streaks</h3>
      <Table
        headers={['Author', 'Total commits', 'Active days', 'Longest streak', 'Current streak']}
        rows={streaks.map((s) => [
          s.email,
          s.totalCommits,
          s.activeDays,
          `${s.longest.start} · ${s.longest.days}d`,
          s.current.days > 0 ? `${s.current.start} · ${s.current.days}d` : '—',
        ])}
      />
    </Card>
  )
}

function ContributorsChart({
  contributors,
}: {
  contributors: import('../lib/types').Contributor[]
}) {
  const data = contributors.slice(0, 10).map((c) => ({
    name: c.name || c.email,
    commits: c.commits,
  }))
  return (
    <Card>
      <h3 className="mb-3 text-sm font-medium text-slate-300">Top contributors</h3>
      <ResponsiveContainer width="100%" height={260}>
        <BarChart data={data} layout="vertical" margin={{ left: 24 }}>
          <XAxis type="number" stroke="#64748b" fontSize={12} />
          <YAxis
            type="category"
            dataKey="name"
            stroke="#64748b"
            fontSize={11}
            width={110}
          />
          <Tooltip
            contentStyle={{ background: '#0f172a', border: '1px solid #334155' }}
          />
          <Bar dataKey="commits" fill="#38bdf8" radius={[0, 4, 4, 0]} />
        </BarChart>
      </ResponsiveContainer>
    </Card>
  )
}

function CommitsView({ repoId }: { repoId: number }) {
  const { data, isLoading } = useCommits(repoId, 200)
  if (isLoading) return <Spinner />
  const commits = data?.commits ?? []
  if (!commits.length) return <Empty message="No commits" />
  return (
    <Table
      headers={['Hash', 'Author', 'Date', 'Message', 'Files', '+', '−']}
      rows={commits.map((c) => [
        <code key="h" className="text-xs text-sky-400">
          {c.hash.slice(0, 8)}
        </code>,
        c.author,
        new Date(c.date).toLocaleDateString(),
        <span key="m" className="max-w-96 truncate">
          {c.message}
        </span>,
        c.filesChanged,
        <span key="i" className="text-emerald-400">
          +{c.insertions}
        </span>,
        <span key="d" className="text-rose-400">
          −{c.deletions}
        </span>,
      ])}
    />
  )
}

function ActivityTab({ repoId }: { repoId: number }) {
  const { data, isLoading } = useHeatmap(repoId)
  if (isLoading) return <Spinner />
  if (!data) return <Empty message="No activity yet" />
  return (
    <div className="flex flex-col gap-4">
      <HeatmapGrid heatmap={data.heatmap} />
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <HourlyHeatmap hourly={data.heatmap.hourly} />
        <StreaksView streaks={data.streaks} />
      </div>
    </div>
  )
}

function ContributorsTab({ repoId }: { repoId: number }) {
  const { data, isLoading } = useContributors(repoId)
  if (isLoading) return <Spinner />
  const contributors = data?.contributors ?? []
  if (!contributors.length) return <Empty message="No contributors" />
  return (
    <div className="flex flex-col gap-4">
      <ContributorsChart contributors={contributors} />
      <Table
        headers={['Name', 'Email', 'Commits', 'Insertions', 'Deletions', 'First commit']}
        rows={contributors.map((c) => [
          c.name,
          c.email,
          c.commits,
          <span key="i" className="text-emerald-400">
            +{c.insertions}
          </span>,
          <span key="d" className="text-rose-400">
            −{c.deletions}
          </span>,
          new Date(c.firstCommitAt).toLocaleDateString(),
        ])}
      />
    </div>
  )
}

function GitInner() {
  const { repoId } = useRepoContext()
  const [tab, setTab] = useState<'activity' | 'commits' | 'contributors'>('activity')
  const repo = useRepo(repoId).data
  if (repoId === 0) return <Empty message="Scan a repository first" />
  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold">{repo?.name ?? ''} · Git</h1>
        <RepoSelector />
      </div>
      <Tabs
        tabs={[
          { id: 'activity', label: 'Activity' },
          { id: 'commits', label: 'Commits' },
          { id: 'contributors', label: 'Contributors' },
        ]}
        active={tab}
        onChange={setTab}
      />
      {tab === 'activity' && <ActivityTab repoId={repoId} />}
      {tab === 'commits' && <CommitsView repoId={repoId} />}
      {tab === 'contributors' && <ContributorsTab repoId={repoId} />}
    </div>
  )
}

export default function Git() {
  return (
    <RepoProvider>
      <GitInner />
    </RepoProvider>
  )
}
