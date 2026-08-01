import { createContext, useContext, useEffect, useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useRepos } from './api'

interface RepoContextValue {
  repos: ReturnType<typeof useRepos>['data'] | undefined
  repoId: number
  setRepoId: (id: number) => void
}

const RepoContext = createContext<RepoContextValue | null>(null)

export function RepoProvider({ children }: { children: React.ReactNode }) {
  const { data } = useRepos()
  const [params, setParams] = useSearchParams()
  const paramId = Number(params.get('repo'))
  const [repoId, setRepoId] = useState(() => {
    const saved = Number(localStorage.getItem('repo-id'))
    return saved > 0 ? saved : 0
  })

  useEffect(() => {
    if (repoId > 0) localStorage.setItem('repo-id', String(repoId))
  }, [repoId])

  useEffect(() => {
    if (paramId > 0) setRepoId(paramId)
  }, [paramId])

  useEffect(() => {
    if (!data?.repositories.length) return
    const exists = data.repositories.some((r) => r.id === repoId)
    if (!exists) setRepoId(data.repositories[0].id)
  }, [data, repoId])

  const value = useMemo(
    () => ({
      repos: data,
      repoId,
      setRepoId: (id: number) => {
        setRepoId(id)
        if (paramId !== id) {
          params.delete('repo')
          setParams(params, { replace: true })
        }
      },
    }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [data, repoId, paramId],
  )
  return <RepoContext.Provider value={value}>{children}</RepoContext.Provider>
}

export function useRepoContext() {
  const ctx = useContext(RepoContext)
  if (!ctx) throw new Error('useRepoContext must be used within RepoProvider')
  return ctx
}
