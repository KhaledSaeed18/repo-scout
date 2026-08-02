import { NavLink, Outlet } from 'react-router-dom'
import { useLiveUpdates } from './lib/ws'
import { cn } from '@/lib/utils'

const nav = [
  { to: '/', label: 'Dashboard' },
  { to: '/scan', label: 'Scan' },
  { to: '/git', label: 'Git' },
  { to: '/files', label: 'Files' },
  { to: '/search', label: 'Search' },
  { to: '/duplicates', label: 'Duplicates' },
  { to: '/architecture', label: 'Architecture' },
  { to: '/metrics', label: 'Metrics' },
  { to: '/dependencies', label: 'Dependencies' },
  { to: '/settings', label: 'Settings' },
]

export default function App() {
  useLiveUpdates()
  return (
    <div className="flex h-full">
      <aside className="flex w-52 flex-col gap-1 border-r border-sidebar-border bg-sidebar p-3">
        <div className="mb-3 px-2 text-sm font-semibold tracking-wide text-sidebar-primary">
          repo-scout
        </div>
        {nav.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            className={({ isActive }) =>
              cn(
                'rounded-md px-3 py-1.5 text-sm transition-colors',
                isActive
                  ? 'bg-sidebar-accent/15 text-sidebar-primary'
                  : 'text-muted-foreground hover:bg-sidebar-accent/10 hover:text-sidebar-foreground',
              )
            }
          >
            {item.label}
          </NavLink>
        ))}
      </aside>
      <main className="flex-1 overflow-auto p-6">
        <Outlet />
      </main>
    </div>
  )
}
