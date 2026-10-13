import React, { Suspense, lazy, useEffect, useMemo } from 'react'
import { createRoot } from 'react-dom/client'
import { useClipare } from './domain/useClipare'
import { WailsBackend } from './domain/wailsBackend'
import { MockBackend } from './preview/mockBackend'
import { PreviewBar } from './preview/PreviewBar'
import type { Platform, View } from './domain/types'
import type { Scenario } from './preview/fixtures'
import './styles/base.css'

const WindowsApp = lazy(() => import('./platform/windows/WindowsApp'))
const MacApp = lazy(() => import('./platform/macos/MacApp'))

function App() {
  const backend = useMemo(() => {
    if (!import.meta.env.DEV) return new WailsBackend()
    const params = new URLSearchParams(location.search)
    const platform: Platform = params.get('platform') === 'macos' ? 'macos' : 'windows'
    const scenario = (params.get('scenario') || 'default') as Scenario
    const theme = params.get('theme') === 'dark' ? 'dark' : 'light'
    return new MockBackend(platform, scenario, theme)
  }, [])
  const { snapshot, error, clearError, perform } = useClipare(backend)
  const [view, setView] = React.useState<View>('home')

  useEffect(() => {
    if (snapshot?.pairing && view === 'home') setView('pairing')
    if (!snapshot?.pairing && view === 'pairing') setView('home')
  }, [snapshot?.pairing, view])

  if (!snapshot) return <main className="loading" role="status">{error || 'Открываем Clipare…'}</main>
  const open = (next: View) => {
    setView(next)
    void perform({ type: 'view.open', view: next })
  }
  return <>
    {import.meta.env.DEV && backend instanceof MockBackend && <PreviewBar backend={backend} snapshot={snapshot} />}
    <Suspense fallback={<main className="loading">Открываем Clipare…</main>}>
      {snapshot.platform === 'windows'
        ? <WindowsApp snapshot={snapshot} view={view} open={open} perform={perform} error={error} clearError={clearError} />
        : <MacApp snapshot={snapshot} view={view} open={open} perform={perform} error={error} clearError={clearError} />}
    </Suspense>
  </>
}

createRoot(document.getElementById('root')!).render(<React.StrictMode><App /></React.StrictMode>)
