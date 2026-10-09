import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import ErrorBoundary from './components/ErrorBoundary'
import './index.css'
import App from './App.tsx'
import Overview from './pages/Overview'
import Activity from './pages/history/Activity'
import Repositories from './pages/Repositories'
import Git from './pages/Git'
import Files from './pages/Files'
import Search from './pages/Search'
import Duplicates from './pages/Duplicates'
import Architecture from './pages/Architecture'
import Metrics from './pages/Metrics'
import Dependencies from './pages/Dependencies'
import Settings from './pages/Settings'

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
              <Route path="/repositories" element={<Repositories />} />
              <Route path="/scan" element={<Navigate to="/repositories" replace />} />
              <Route path="/activity" element={<Activity />} />
              <Route path="/git" element={<Git />} />
              <Route path="/files" element={<Files />} />
              <Route path="/search" element={<Search />} />
              <Route path="/duplicates" element={<Duplicates />} />
              <Route path="/architecture" element={<Architecture />} />
              <Route path="/metrics" element={<Metrics />} />
              <Route path="/dependencies" element={<Dependencies />} />
              <Route path="/settings" element={<Settings />} />
            </Route>
          </Routes>
        </BrowserRouter>
      </QueryClientProvider>
    </ErrorBoundary>
  </StrictMode>,
)
