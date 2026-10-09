import { FolderGit2, Radar } from 'lucide-react'
import { Fragment, type ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { buttonVariants, Progress } from '@/components/ui'
import { useJobs } from '@/lib/api'
import { useRepoContext } from '@/lib/repo-context'
import type { Repository } from '@/lib/types'
import { Empty, Loading } from './states'

function FirstScan({ repo }: { repo: Repository }) {
  const { data } = useJobs()
  const job = data?.jobs.find((j) => j.repoId === repo.id && (j.status === 'running' || j.status === 'queued'))
  const pct = Math.round((job?.progress ?? 0) * 100)
  return (
    <div className="max-w-md py-10">
      <Radar className="mb-3 size-5 text-primary" aria-hidden />
      <p className="font-medium">Scanning {repo.name}</p>
      <p className="mb-4 text-sm text-muted-foreground">
        {job?.message ? `Working on ${job.message}.` : 'Waiting for a worker.'} This page fills in when the
        scan finishes.
      </p>
      <Progress value={pct} aria-label="Scan progress" />
    </div>
  )
}

/**
 * Gate for pages that inspect a repository. Shows an invitation when there
 * are none, scan progress during the first scan, and otherwise renders the
 * page for the selected repository.
 */
export default function RequireRepo({ children }: { children: (repo: Repository) => ReactNode }) {
  const { repo, isLoading } = useRepoContext()
  if (isLoading) return <Loading label="Loading repositories…" />
  if (!repo) {
    return (
      <Empty icon={FolderGit2} title="No repositories yet" className="mt-4">
        <p className="mb-3">Add a local Git repository to see its history, code and structure.</p>
        <Link to="/repositories" className={buttonVariants()}>
          Add a repository
        </Link>
      </Empty>
    )
  }
  // Never scanned yet: either waiting for a worker or scanning right now.
  if (!repo.lastScannedAt && repo.status !== 'failed') return <FirstScan repo={repo} />
  if (!repo.lastScannedAt && repo.status === 'failed') {
    return (
      <Empty icon={FolderGit2} title={`The scan of ${repo.name} failed`} className="mt-4">
        <p className="mb-3">Check that the folder still exists and is readable, then scan it again.</p>
        <Link to="/repositories" className={buttonVariants({ variant: 'outline' })}>
          Go to repositories
        </Link>
      </Empty>
    )
  }
  // Keyed by repository so page state (filters, paging, selections) starts
  // fresh when the selection changes.
  return <Fragment key={repo.id}>{children(repo)}</Fragment>
}
