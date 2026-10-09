import { Menu, X } from 'lucide-react'
import { Suspense, useEffect, useState } from 'react'
import { Outlet, useLocation } from 'react-router-dom'
import { Button } from '@/components/ui'
import BrandMark from './components/BrandMark'
import ErrorBoundary from './components/ErrorBoundary'
import RepoProvider from './components/RepoProvider'
import Sidebar from './components/Sidebar'
import ThemeSync from './components/ThemeSync'
import { Loading } from './components/states'
import { useLiveUpdates } from './lib/ws'

export default function App() {
  useLiveUpdates()
  const [menuOpen, setMenuOpen] = useState(false)
  const { pathname } = useLocation()
  useEffect(() => setMenuOpen(false), [pathname])
  useEffect(() => {
    if (!menuOpen) return
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && setMenuOpen(false)
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [menuOpen])

  return (
    <RepoProvider>
      <ThemeSync />
      <div className="flex h-full">
        <aside className="hidden w-60 shrink-0 border-r border-sidebar-border bg-sidebar lg:block">
          <Sidebar />
        </aside>

        {menuOpen && (
          <div className="fixed inset-0 z-40 lg:hidden">
            <button
              aria-label="Close menu"
              className="absolute inset-0 bg-foreground/30"
              onClick={() => setMenuOpen(false)}
            />
            <aside aria-label="Menu" className="relative h-full w-64 border-r border-sidebar-border bg-sidebar">
              <Sidebar onNavigate={() => setMenuOpen(false)} />
            </aside>
          </div>
        )}

        <div className="flex min-w-0 flex-1 flex-col">
          <header className="flex items-center gap-2 border-b bg-sidebar px-4 py-2 lg:hidden">
            <Button variant="ghost" size="icon" aria-label="Open menu" onClick={() => setMenuOpen(true)}>
              {menuOpen ? <X /> : <Menu />}
            </Button>
            <BrandMark className="size-6" />
            <span className="font-heading font-semibold">Repo Scout</span>
          </header>
          <main className="flex-1 overflow-y-auto">
            <div className="mx-auto w-full max-w-6xl px-5 py-8 sm:px-8">
              <ErrorBoundary resetKey={pathname}>
                <Suspense fallback={<Loading label="Loading page…" />}>
                  <Outlet />
                </Suspense>
              </ErrorBoundary>
            </div>
          </main>
        </div>
      </div>
    </RepoProvider>
  )
}
