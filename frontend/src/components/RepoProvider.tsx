import { useEffect, useMemo, useState, type ReactNode } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useRepos } from '@/lib/api'
import { RepoContext } from '@/lib/repo-context'

const storageKey = 'repo-scout-repo'

/**
 * Owns the app-wide repository selection. A `?repo=<id>` link selects that
 * repository once and is then removed from the URL; the choice persists in
 * localStorage. Missing or deleted selections fall back to the first repo.
 */
export default function RepoProvider({ children }: { children: ReactNode }) {
  const { data, isLoading } = useRepos()
  const [params, setParams] = useSearchParams()
  const [selected, setSelected] = useState(() => Number(localStorage.getItem(storageKey)) || 0)

  const linked = Number(params.get('repo')) || 0
  useEffect(() => {
    if (!linked) return
    setSelected(linked)
    const next = new URLSearchParams(params)
    next.delete('repo')
    setParams(next, { replace: true })
  }, [linked, params, setParams])

  useEffect(() => {
    if (selected > 0) localStorage.setItem(storageKey, String(selected))
  }, [selected])

  const value = useMemo(() => {
    const repos = data?.repositories ?? []
    const repo = repos.find((r) => r.id === selected) ?? repos[0]
    return { repos, isLoading, repo, repoId: repo?.id ?? 0, setRepoId: setSelected }
  }, [data, isLoading, selected])

  return <RepoContext.Provider value={value}>{children}</RepoContext.Provider>
}
