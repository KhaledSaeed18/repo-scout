import { lazy, StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import ErrorBoundary from './components/ErrorBoundary'
import './index.css'
import App from './App.tsx'

// Pages load on demand so the first paint does not wait for charts and graphs.
const Overview = lazy(() => import('./pages/Overview'))
const Repositories = lazy(() => import('./pages/Repositories'))
const Portfolio = lazy(() => import('./pages/Portfolio'))
const Activity = lazy(() => import('./pages/history/Activity'))
const Commits = lazy(() => import('./pages/history/Commits'))
const Contributors = lazy(() => import('./pages/history/Contributors'))
const Refs = lazy(() => import('./pages/history/Refs'))
const Knowledge = lazy(() => import('./pages/history/Knowledge'))
const Files = lazy(() => import('./pages/code/Files'))
const Search = lazy(() => import('./pages/code/Search'))
const Metrics = lazy(() => import('./pages/code/Metrics'))
const Duplicates = lazy(() => import('./pages/code/Duplicates'))
const Hotspots = lazy(() => import('./pages/code/Hotspots'))
const Architecture = lazy(() => import('./pages/structure/Architecture'))
const Dependencies = lazy(() => import('./pages/structure/Dependencies'))
const Coupling = lazy(() => import('./pages/structure/Coupling'))
const Settings = lazy(() => import('./pages/Settings'))
const NotFound = lazy(() => import('./pages/NotFound'))

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      staleTime: 5000,
      refetchOnWindowFocus: false,
    },
  },
})

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ErrorBoundary>
      <QueryClientProvider client={queryClient}>
        <BrowserRouter>
          <Routes>
            <Route element={<App />}>
              <Route path="/" element={<Overview />} />
              <Route path="/portfolio" element={<Portfolio />} />
              <Route path="/repositories" element={<Repositories />} />
              <Route path="/activity" element={<Activity />} />
              <Route path="/commits" element={<Commits />} />
              <Route path="/contributors" element={<Contributors />} />
              <Route path="/knowledge" element={<Knowledge />} />
              <Route path="/branches" element={<Refs />} />
              <Route path="/files" element={<Files />} />
              <Route path="/search" element={<Search />} />
              <Route path="/metrics" element={<Metrics />} />
              <Route path="/hotspots" element={<Hotspots />} />
              <Route path="/duplicates" element={<Duplicates />} />
              <Route path="/architecture" element={<Architecture />} />
              <Route path="/coupling" element={<Coupling />} />
              <Route path="/dependencies" element={<Dependencies />} />
              <Route path="/settings" element={<Settings />} />
              {/* Old addresses from before the redesign. */}
              <Route path="/scan" element={<Navigate to="/repositories" replace />} />
              <Route path="/git" element={<Navigate to="/activity" replace />} />
              <Route path="*" element={<NotFound />} />
            </Route>
          </Routes>
        </BrowserRouter>
      </QueryClientProvider>
    </ErrorBoundary>
  </StrictMode>,
)
