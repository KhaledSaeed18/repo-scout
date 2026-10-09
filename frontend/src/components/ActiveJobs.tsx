import { Link } from 'react-router-dom'
import { Progress } from '@/components/ui'
import { useJobs } from '@/lib/api'
import { useRepoContext } from '@/lib/repo-context'

const active = new Set(['queued', 'running', 'paused'])

/** Compact live status of scans in progress, shown in the sidebar. */
export default function ActiveJobs() {
  const { data } = useJobs()
  const { repos } = useRepoContext()
  const jobs = (data?.jobs ?? []).filter((j) => active.has(j.status))
  if (!jobs.length) return null

  return (
    <Link to="/repositories" className="block space-y-2 rounded-md border bg-card p-3 text-sm hover:border-primary/50">
      {jobs.slice(0, 3).map((job) => {
        const name = repos.find((r) => r.id === job.repoId)?.name ?? `Repository ${job.repoId}`
        return (
          <div key={job.id} className="space-y-1.5">
            <div className="flex justify-between gap-2">
              <span className="truncate font-medium">{name}</span>
              <span className="shrink-0 text-muted-foreground tabular-nums">
                {job.status === 'running' ? `${Math.round(job.progress * 100)}%` : job.status}
              </span>
            </div>
            <Progress value={Math.round(job.progress * 100)} aria-label={`Scanning ${name}`} />
          </div>
        )
      })}
    </Link>
  )
}
