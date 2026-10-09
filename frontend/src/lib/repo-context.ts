import { createContext, useContext } from 'react'
import type { Repository } from './types'

export interface RepoContextValue {
  /** Every known repository, most recently updated first. */
  repos: Repository[]
  /** True until the repository list has loaded once. */
  isLoading: boolean
  /** The selected repository, if any exist. */
  repo: Repository | undefined
  repoId: number
  setRepoId: (id: number) => void
}

export const RepoContext = createContext<RepoContextValue | null>(null)

export function useRepoContext(): RepoContextValue {
  const ctx = useContext(RepoContext)
  if (!ctx) throw new Error('useRepoContext must be used within RepoProvider')
  return ctx
}
