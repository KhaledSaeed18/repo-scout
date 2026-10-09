import { Activity, Flame, FolderGit2, GitBranch, Tag } from 'lucide-react'
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
import { Card, CardContent, CardHeader, CardTitle, EmptyState, Spinner, Tabs, TabsContent, TabsList, TabsTrigger, Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui'
import { useHeatmap, useRepo, useBranches, useTags } from '../lib/api'
import { useRepoContext } from '@/lib/repo-context'
import RepoSelector from '../components/RepoSelector'

function HeatmapGrid({ heatmap }: { heatmap: import('../lib/types').Heatmap }) {
  const daily = heatmap.daily ?? []
  const max = Math.max(1, ...daily.map((d) => d.count))
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
    if (count === 0) return 'bg-muted'
    if (r > 0.66) return 'bg-chart-2'
    if (r > 0.33) return 'bg-chart-2/60'
    return 'bg-chart-2/30'
  }
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">
          Commit activity ({heatmap.total} commits, {heatmap.start} → {heatmap.end})
        </CardTitle>
      </CardHeader>
      <CardContent className="flex gap-1 overflow-x-auto">
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
      </CardContent>
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
      <CardHeader>
        <CardTitle className="text-sm">Hour of week</CardTitle>
      </CardHeader>
      <CardContent>
        <ResponsiveContainer width="100%" height={180}>
          <BarChart data={data}>
            <XAxis dataKey="hr" hide />
            <YAxis hide />
            <Tooltip
              contentStyle={{ background: 'var(--popover)', border: '1px solid var(--border)' }}
              labelStyle={{ color: 'var(--popover-foreground)' }}
              formatter={(value) => [Number(value), 'commits']}
              labelFormatter={(_, p) => {
                const d = p?.[0]?.payload as { wd: number; hr: number }
                return d ? `weekday ${d.wd} · ${d.hr}:00` : ''
              }}
            />
            <Bar dataKey="count" radius={[2, 2, 0, 0]} isAnimationActive={false}>
              {data.map((d, i) => (
                <Cell key={i} fill="var(--chart-1)" fillOpacity={d.count / max > 0.5 ? 1 : 0.5} />
              ))}
            </Bar>
          </BarChart>
        </ResponsiveContainer>
      </CardContent>
    </Card>
  )
}

function StreaksView({ streaks }: { streaks: import('../lib/types').StreaksResult[] }) {
  if (!streaks || !streaks.length) {
    return (
      <Card>
        <CardContent>
          <EmptyState icon={Flame} title="No streak data" />
        </CardContent>
      </Card>
    )
  }
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">Streaks</CardTitle>
      </CardHeader>
      <CardContent className="overflow-x-auto">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Author</TableHead>
              <TableHead>Total commits</TableHead>
              <TableHead>Active days</TableHead>
              <TableHead>Longest streak</TableHead>
              <TableHead>Current streak</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {streaks.map((s) => (
              <TableRow key={s.email}>
                <TableCell>{s.email}</TableCell>
                <TableCell>{s.totalCommits}</TableCell>
                <TableCell>{s.activeDays}</TableCell>
                <TableCell>{`${s.longest.start} · ${s.longest.days}d`}</TableCell>
                <TableCell>{s.current.days > 0 ? `${s.current.start} · ${s.current.days}d` : '—'}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  )
}

function BranchesTagsView({ repoId }: { repoId: number }) {
  const { data: branchesData, isLoading: branchesLoading } = useBranches(repoId)
  const { data: tagsData, isLoading: tagsLoading } = useTags(repoId)
  const branches = branchesData?.branches ?? []
  const tags = tagsData?.tags ?? []
  if (branchesLoading || tagsLoading) return <Spinner />
  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardHeader>
          <CardTitle className="text-sm">Branches</CardTitle>
        </CardHeader>
        <CardContent>
          {branches.length === 0 ? (
            <EmptyState icon={GitBranch} title="No branches" />
          ) : (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Commit</TableHead>
                    <TableHead>Current</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {branches.map((b) => (
                    <TableRow key={b.name}>
                      <TableCell><span className={b.isCurrent ? 'text-primary font-medium' : ''}>{b.name}</span></TableCell>
                      <TableCell><code className="text-xs text-muted-foreground">{b.commitHash.slice(0, 8)}</code></TableCell>
                      <TableCell>{b.isCurrent ? <span className="text-chart-2 text-xs">HEAD</span> : ''}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle className="text-sm">Tags</CardTitle>
        </CardHeader>
        <CardContent>
          {tags.length === 0 ? (
            <EmptyState icon={Tag} title="No tags" />
          ) : (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Commit</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {tags.map((t) => (
                    <TableRow key={t.name}>
                      <TableCell>{t.name}</TableCell>
                      <TableCell><code className="text-xs text-muted-foreground">{t.commitHash.slice(0, 8)}</code></TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

function ActivityTab({ repoId }: { repoId: number }) {
  const { data, isLoading } = useHeatmap(repoId)
  if (isLoading) return <Spinner />
  if (!data) return <EmptyState icon={Activity} title="No activity yet" />
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

export default function Git() {
  const { repoId } = useRepoContext()
  const [tab, setTab] = useState<'activity' | 'branches-tags'>('activity')
  const repo = useRepo(repoId).data
  if (repoId === 0) return <EmptyState icon={FolderGit2} title="Scan a repository first" />
  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold">{repo?.name ?? ''} · Git</h1>
        <RepoSelector />
      </div>
      <Tabs value={tab} onValueChange={setTab}>
        <TabsList className="w-full justify-start">
          <TabsTrigger value="activity">Activity</TabsTrigger>
          <TabsTrigger value="branches-tags">Branches & Tags</TabsTrigger>
        </TabsList>
        <TabsContent value="activity"><ActivityTab repoId={repoId} /></TabsContent>
        <TabsContent value="branches-tags"><BranchesTagsView repoId={repoId} /></TabsContent>
      </Tabs>
    </div>
  )
}
