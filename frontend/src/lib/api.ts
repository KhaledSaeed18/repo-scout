import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import type {
  Architecture,
  Branch,
  Commit,
  Contributor,
  DependenciesResponse,
  DuplicatesResponse,
  FileEntry,
  HeatmapResponse,
  Job,
  LargestCommit,
  Metrics,
  Ownership,
  RepoListResponse,
  Repository,
  SearchResult,
  Settings,
  Tag,
  TreeResponse,
} from './types'

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    headers: { 'Content-Type': 'application/json' },
    ...init,
  })
  if (!res.ok) {
    let detail = res.statusText
    try {
      const body = await res.json()
      detail = body.error ?? detail
    } catch {
      /* ignore */
    }
    throw new Error(detail)
  }
  if (res.status === 204) return undefined as T
  return res.json() as Promise<T>
}

const get = <T>(path: string) => request<T>(path)

export const api = {
  health: () => get<{ status: string }>('/api/health'),
  listRepos: () => get<RepoListResponse>('/api/repositories'),
  getRepo: (id: number) => get<Repository>(`/api/repositories/${id}`),
  createRepo: (path: string) =>
    request<{ repository: Repository; job: Job }>('/api/repositories', {
      method: 'POST',
      body: JSON.stringify({ path }),
    }),
  deleteRepo: (id: number) =>
    request<{ deleted: boolean }>(`/api/repositories/${id}`, { method: 'DELETE' }),
  files: (id: number, params: Record<string, string | number>) => {
    const qs = new URLSearchParams(
      Object.entries(params).map(([k, v]) => [k, String(v)]),
    ).toString()
    return get<{ files: FileEntry[]; total: number }>(
      `/api/repositories/${id}/files?${qs}`,
    )
  },
  tree: (id: number, folder: string) =>
    get<TreeResponse>(`/api/repositories/${id}/tree?folder=${encodeURIComponent(folder)}`),
  commits: (id: number, limit = 100) =>
    get<{ commits: Commit[] }>(`/api/repositories/${id}/commits?limit=${limit}`),
  contributors: (id: number) =>
    get<{ contributors: Contributor[] }>(`/api/repositories/${id}/contributors`),
  largestCommits: (id: number) =>
    get<{ commits: LargestCommit[] }>(`/api/repositories/${id}/largest-commits`),
  ownership: (id: number) =>
    get<Ownership>(`/api/repositories/${id}/ownership`),
  branches: (id: number) =>
    get<{ branches: Branch[] }>(`/api/repositories/${id}/branches`),
  tags: (id: number) =>
    get<{ tags: Tag[] }>(`/api/repositories/${id}/tags`),
  heatmap: (id: number) => get<HeatmapResponse>(`/api/repositories/${id}/heatmap`),
  dependencies: (id: number) => get<DependenciesResponse>(`/api/repositories/${id}/dependencies`),
  duplicates: (id: number) => get<DuplicatesResponse>(`/api/repositories/${id}/duplicates`),
  architecture: (id: number) => get<Architecture>(`/api/repositories/${id}/architecture`),
  metrics: (id: number) => get<Metrics>(`/api/repositories/${id}/metrics`),
  search: (params: Record<string, string | number | boolean>) => {
    const qs = new URLSearchParams(
      Object.entries(params).map(([k, v]) => [k, String(v)]),
    ).toString()
    return get<SearchResult>(`/api/search?${qs}`)
  },
  jobs: () => get<{ jobs: Job[] }>('/api/jobs'),
  jobAction: (id: number, action: 'pause' | 'resume' | 'cancel') =>
    request<{ ok: boolean }>(`/api/jobs/${id}/${action}`, { method: 'POST' }),
  settings: () => get<Settings>('/api/settings'),
  saveSettings: (settings: Settings) =>
    request<Settings>('/api/settings', { method: 'PUT', body: JSON.stringify(settings) }),
  exportUrl: (id: number, kind: string, format: string) =>
    `/api/repositories/${id}/export?kind=${kind}&format=${format}`,
  svgUrl: (id: number) => `/api/repositories/${id}/svg`,
}

export const wsUrl = () => {
  const proto = window.location.protocol === 'https:' ? 'wss' : 'ws'
  return `${proto}://localhost:8080/api/ws`
}

export const useRepos = () =>
  useQuery({ queryKey: ['repos'], queryFn: () => api.listRepos() })

export const useRepo = (id: number) =>
  useQuery({
    queryKey: ['repo', id],
    queryFn: () => api.getRepo(id),
    enabled: id > 0,
  })

export const useFiles = (id: number, params: Record<string, string | number>) =>
  useQuery({
    queryKey: ['files', id, params],
    queryFn: () => api.files(id, params),
    enabled: id > 0,
  })

export const useTree = (id: number, folder: string) =>
  useQuery({
    queryKey: ['tree', id, folder],
    queryFn: () => api.tree(id, folder),
    enabled: id > 0,
  })

export const useCommits = (id: number, limit = 100) =>
  useQuery({
    queryKey: ['commits', id],
    queryFn: () => api.commits(id, limit),
    enabled: id > 0,
  })

export const useContributors = (id: number) =>
  useQuery({
    queryKey: ['contributors', id],
    queryFn: () => api.contributors(id),
    enabled: id > 0,
  })

export const useLargestCommits = (id: number) =>
  useQuery({
    queryKey: ['largestCommits', id],
    queryFn: () => api.largestCommits(id),
    enabled: id > 0,
  })

export const useOwnership = (id: number) =>
  useQuery({
    queryKey: ['ownership', id],
    queryFn: () => api.ownership(id),
    enabled: id > 0,
  })

export const useBranches = (id: number) =>
  useQuery({
    queryKey: ['branches', id],
    queryFn: () => api.branches(id),
    enabled: id > 0,
  })

export const useTags = (id: number) =>
  useQuery({
    queryKey: ['tags', id],
    queryFn: () => api.tags(id),
    enabled: id > 0,
  })

export const useHeatmap = (id: number) =>
  useQuery({
    queryKey: ['heatmap', id],
    queryFn: () => api.heatmap(id),
    enabled: id > 0,
  })

export const useDependencies = (id: number) =>
  useQuery({
    queryKey: ['dependencies', id],
    queryFn: () => api.dependencies(id),
    enabled: id > 0,
  })

export const useDuplicates = (id: number) =>
  useQuery({
    queryKey: ['duplicates', id],
    queryFn: () => api.duplicates(id),
    enabled: id > 0,
  })

export const useArchitecture = (id: number) =>
  useQuery({
    queryKey: ['architecture', id],
    queryFn: () => api.architecture(id),
    enabled: id > 0,
  })

export const useMetrics = (id: number) =>
  useQuery({
    queryKey: ['metrics', id],
    queryFn: () => api.metrics(id),
    enabled: id > 0,
  })

export const useJobs = (refetchMs = 3000) =>
  useQuery({ queryKey: ['jobs'], queryFn: () => api.jobs(), refetchInterval: refetchMs })

export const useSettings = () =>
  useQuery({ queryKey: ['settings'], queryFn: () => api.settings() })

export const useCreateRepo = () => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (path: string) => api.createRepo(path),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['repos'] })
    },
  })
}

export const useDeleteRepo = () => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => api.deleteRepo(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['repos'] })
    },
  })
}

export const useSaveSettings = () => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (s: Settings) => api.saveSettings(s),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['settings'] })
    },
  })
}
