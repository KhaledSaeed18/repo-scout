import {
  CalendarDays,
  Copy,
  FolderGit2,
  FolderTree,
  Gauge,
  GitBranch,
  GitCommitHorizontal,
  Map as MapIcon,
  Package,
  Search,
  Settings2,
  Users,
  Waypoints,
  type LucideIcon,
} from 'lucide-react'
import { NavLink } from 'react-router-dom'
import { cn } from '@/lib/utils'
import ActiveJobs from './ActiveJobs'
import RepoSwitcher from './RepoSwitcher'

interface NavItem {
  to: string
  label: string
  icon: LucideIcon
}

/** Pages grouped by the part of the repository they inspect. */
const groups: { label?: string; items: NavItem[] }[] = [
  { items: [{ to: '/', label: 'Overview', icon: MapIcon }] },
  {
    label: 'History',
    items: [
      { to: '/activity', label: 'Activity', icon: CalendarDays },
      { to: '/commits', label: 'Commits', icon: GitCommitHorizontal },
      { to: '/contributors', label: 'Contributors', icon: Users },
      { to: '/branches', label: 'Branches & tags', icon: GitBranch },
    ],
  },
  {
    label: 'Code',
    items: [
      { to: '/files', label: 'Files', icon: FolderTree },
      { to: '/search', label: 'Search', icon: Search },
      { to: '/metrics', label: 'Metrics', icon: Gauge },
      { to: '/duplicates', label: 'Duplicates', icon: Copy },
    ],
  },
  {
    label: 'Structure',
    items: [
      { to: '/architecture', label: 'Architecture', icon: Waypoints },
      { to: '/dependencies', label: 'Dependencies', icon: Package },
    ],
  },
]

const footer: NavItem[] = [
  { to: '/repositories', label: 'Repositories', icon: FolderGit2 },
  { to: '/settings', label: 'Settings', icon: Settings2 },
]

function Item({ item, onNavigate }: { item: NavItem; onNavigate?: () => void }) {
  const Icon = item.icon
  return (
    <NavLink
      to={item.to}
      end={item.to === '/'}
      onClick={onNavigate}
      className={({ isActive }) =>
        cn(
          'flex items-center gap-2.5 rounded-md px-2.5 py-1.5 text-sm transition-colors',
          isActive
            ? 'bg-card font-medium text-foreground shadow-[inset_2px_0_0_var(--primary)]'
            : 'text-muted-foreground hover:bg-sidebar-accent hover:text-foreground',
        )
      }
    >
      <Icon className="size-4 shrink-0" aria-hidden />
      {item.label}
    </NavLink>
  )
}

/** Compass glyph drawn in theme colors. */
function Mark() {
  return (
    <svg viewBox="0 0 24 24" className="size-6" aria-hidden>
      <circle cx="12" cy="12" r="10" fill="none" stroke="currentColor" strokeWidth="1.75" />
      <path d="M16.5 7.5 13.4 13.4 7.5 16.5l3.1-5.9z" fill="var(--primary)" />
      <circle cx="12" cy="12" r="1.25" fill="var(--sidebar)" />
    </svg>
  )
}

export default function Sidebar({ onNavigate }: { onNavigate?: () => void }) {
  return (
    <div className="flex h-full flex-col gap-5 overflow-y-auto px-3 py-4">
      <div className="flex items-center gap-2 px-2.5">
        <Mark />
        <span className="font-heading text-lg font-semibold tracking-tight">Repo Scout</span>
      </div>

      <RepoSwitcher />

      <nav aria-label="Repository" className="flex flex-col gap-4">
        {groups.map((group, i) => (
          <div key={group.label ?? i} className="flex flex-col gap-0.5">
            {group.label && (
              <span className="px-2.5 pb-1 text-xs font-medium text-muted-foreground/80">{group.label}</span>
            )}
            {group.items.map((item) => (
              <Item key={item.to} item={item} onNavigate={onNavigate} />
            ))}
          </div>
        ))}
      </nav>

      <div className="mt-auto flex flex-col gap-3">
        <ActiveJobs />
        <nav aria-label="App" className="flex flex-col gap-0.5 border-t pt-3">
          {footer.map((item) => (
            <Item key={item.to} item={item} onNavigate={onNavigate} />
          ))}
        </nav>
      </div>
    </div>
  )
}
