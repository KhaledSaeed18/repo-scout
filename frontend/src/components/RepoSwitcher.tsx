import { Plus } from 'lucide-react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { Select, SelectContent, SelectItem, SelectSeparator, SelectTrigger, SelectValue } from '@/components/ui'
import { useRepoContext } from '@/lib/repo-context'

/** Picks the repository every page in the app is looking at. */
export default function RepoSwitcher() {
  const { repos, repo, setRepoId } = useRepoContext()
  const { pathname } = useLocation()
  const navigate = useNavigate()
  // Query params such as ?file= describe the previous repository.
  const switchTo = (id: number) => {
    setRepoId(id)
    navigate(pathname, { replace: true })
  }

  if (!repo) {
    return (
      <Link
        to="/repositories"
        className="flex items-center gap-2 rounded-md border border-dashed border-input px-3 py-2 text-sm text-muted-foreground hover:border-primary hover:text-primary"
      >
        <Plus className="size-4" />
        Add a repository
      </Link>
    )
  }

  const items = repos.map((r) => ({ value: String(r.id), label: r.name }))
  return (
    <Select items={items} value={String(repo.id)} onValueChange={(v) => v && switchTo(Number(v))}>
      <SelectTrigger aria-label="Repository" className="w-full bg-transparent px-2.5 py-2 hover:bg-sidebar-accent data-[size=default]:h-auto">
        <SelectValue>
          {() => (
            <span className="flex min-w-0 flex-col items-start gap-0.5 leading-tight">
              <span className="w-full truncate font-heading text-base font-semibold">{repo.name}</span>
              <span className="w-full truncate text-xs text-muted-foreground">
                {repo.defaultBranch || 'no branch'}
              </span>
            </span>
          )}
        </SelectValue>
      </SelectTrigger>
      <SelectContent alignItemWithTrigger={false} className="min-w-64">
        {repos.map((r) => (
          <SelectItem key={r.id} value={String(r.id)}>
            <span className="flex min-w-0 flex-col">
              <span className="truncate font-medium">{r.name}</span>
              <span className="truncate text-xs text-muted-foreground">{r.path}</span>
            </span>
          </SelectItem>
        ))}
        <SelectSeparator />
        <Link
          to="/repositories"
          className="flex items-center gap-2 rounded-md px-1.5 py-1.5 text-sm text-primary hover:bg-accent"
        >
          <Plus className="size-4" />
          Add a repository
        </Link>
      </SelectContent>
    </Select>
  )
}
