export interface Repository {
  id: number
  name: string
  path: string
  status: 'scanning' | 'ready' | 'failed'
  gitRemote: string
  headCommit: string
  defaultBranch: string
  lastScannedAt: string | null
  fileCount: number
  totalLOC: number
  totalCode: number
  totalComments: number
  totalBlank: number
  totalSize: number
  commitCount: number
  contributorCount: number
  dependencyCount: number
  dupGroupCount: number
  createdAt: string
  updatedAt: string
}

export interface FileEntry {
  id: number
  repoId: number
  path: string
  name: string
  folder: string
  extension: string
  language: string
  size: number
  linesTotal: number
  linesCode: number
  linesComment: number
  linesBlank: number
  complexity: number
  imports: number
  exports: number
  author: string
  firstCommitAt: string | null
  lastCommitAt: string | null
  commits: number
  funcCount: number
  avgFuncLen: number
  maxFuncLen: number
  avgNesting: number
  maxNesting: number
}

export interface Commit {
  id: number
  repoId: number
  hash: string
  author: string
  email: string
  date: string
  message: string
  filesChanged: number
  insertions: number
  deletions: number
  isMerge: boolean
}

export interface Contributor {
  id: number
  repoId: number
  name: string
  email: string
  commits: number
  insertions: number
  deletions: number
  firstCommitAt: string
  lastCommitAt: string
}

export interface DayActivity {
  date: string
  count: number
}

export interface Heatmap {
  daily: DayActivity[]
  hourly: number[][]
  total: number
  start: string
  end: string
}

export interface Streak {
  start: string
  end: string
  days: number
}

export interface StreaksResult {
  email: string
  longest: Streak
  current: Streak
  all: Streak[]
  activeDays: number
  totalCommits: number
}

export interface Dependency {
  id: number
  repoId: number
  filePath: string
  manager: string
  name: string
  version: string
  scope: string
}

export interface DuplicateBlock {
  id: number
  groupId: number
  repoId: number
  filePath: string
  startLine: number
  endLine: number
}

export interface DuplicateGroup {
  id: number
  repoId: number
  fragment: string
  lines: number
  similarity: number
  fileCount: number
  blocks: DuplicateBlock[]
}

export interface ArchEdge {
  from: string
  to: string
  kind: string
  resolved: boolean
}

export interface Architecture {
  edges: ArchEdge[]
  cycles: string[][]
  unusedModules: string[]
  deadFiles: string[]
  entryPoints: string[]
  unresolved: ArchEdge[]
  folders: string[]
}

export interface MetricsTotals {
  files: number
  loc: number
  code: number
  comments: number
  blank: number
  complexity: number
  funcs: number
  imports: number
  exports: number
}

export interface Metrics {
  totals: MetricsTotals
  languages: Record<string, { files: number; loc: number; code: number }>
  maxDepth: number
  deepestFile: string
  largestFiles: FileEntry[]
  mostComplexFiles: FileEntry[]
}

export interface SearchMatch {
  line: number
  text: string
}

export interface SearchHit {
  fileId: number
  path: string
  language: string
  size: number
  linesTotal: number
  matches: SearchMatch[]
}

export interface SearchResult {
  hits: SearchHit[]
  total: number
  truncated: boolean
}

export interface Settings {
  ignoreFolders: string[]
  ignoreExtensions: string[]
  maxFileSize: number
  maxFileDepth: number
  workerCount: number
  maxSearchFiles: number
  dupMinLines: number
  dupMinSimilarity: number
  theme: string
}

export interface Job {
  id: number
  repoId: number
  kind: string
  status: string
  progress: number
  current: number
  total: number
  message: string
  error: string
  createdAt: string
  updatedAt: string
  startedAt: string | null
  finishedAt: string | null
}

export interface TreeResponse {
  folder: string
  folders: string[]
  files: FileEntry[]
}

export interface RepoListResponse {
  repositories: Repository[]
}

export interface HeatmapResponse {
  heatmap: Heatmap
  streaks: StreaksResult[]
}

export interface DependenciesResponse {
  managers: string[]
  dependencies: Record<string, Dependency[]>
  total: number
}

export interface LargestCommit {
  hash: string
  author: string
  email: string
  date: string
  message: string
  filesChanged: number
  insertions: number
  deletions: number
}

export interface Ownership {
  byAuthor: OwnerSummary[]
  total: number
}

export interface OwnerSummary {
  author: string
  files: number
  share: number
}

export interface DuplicatesResponse {
  groups: DuplicateGroup[]
  total: number
}

export interface Branch {
  id: number
  repoId: number
  name: string
  commitHash: string
  isCurrent: boolean
}

export interface Tag {
  id: number
  repoId: number
  name: string
  commitHash: string
}

export interface BrowseEntry {
  name: string
  path: string
}

export interface BrowseResponse {
  path: string
  parent: string
  entries: BrowseEntry[]
}
