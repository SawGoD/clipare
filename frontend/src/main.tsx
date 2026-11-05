import React, { useEffect, useMemo } from 'react'
import { createRoot } from 'react-dom/client'
import { useClipare } from './domain/useClipare'
import { WailsBackend } from './domain/wailsBackend'
import { MockBackend } from './preview/mockBackend'
import { PreviewBar } from './preview/PreviewBar'
import type { Platform, View } from './domain/types'
import type { Scenario } from './preview/fixtures'
import WindowsApp from './platform/windows/WindowsApp'
import MacApp from './platform/macos/MacApp'
import { RenderBoundary } from './RenderBoundary'
import './styles/base.css'

function App() {
  const backend = useMemo(() => {
    if (!import.meta.env.DEV) return new WailsBackend()
    const params = new URLSearchParams(location.search)
    const platform: Platform = params.get('platform') === 'macos' ? 'macos' : 'windows'
    const scenario = (params.get('scenario') || 'default') as Scenario
    const theme = params.get('theme') === 'dark' ? 'dark' : 'light'
    return new MockBackend(platform, scenario, theme)
  }, [])
  const { snapshot, error, retrySnapshot, clearError, perform } = useClipare(backend)
  const [view, setView] = React.useState<View>('home')

  useEffect(() => {
    if (snapshot?.view) setView(snapshot.view)
  }, [snapshot?.view])

  if (import.meta.env.DEV && new URLSearchParams(location.search).get('scenario') === 'render-error') {
    throw new Error('Preview rendering failure')
  }
  if (!snapshot) return <main className="recovery-screen" role={error ? 'alert' : 'status'}>
    <div className="recovery-card"><h1>Clipare</h1><p>{error || 'Открываем Clipare…'}</p>
      {error && <button type="button" onClick={retrySnapshot}>Повторить</button>}
    </div>
  </main>
  const open = (next: View) => {
    setView(next)
    if (next !== 'pairing') void perform({ type: 'view.open', view: next })
  }
  return <>
    {import.meta.env.DEV && backend instanceof MockBackend && <PreviewBar backend={backend} snapshot={snapshot} />}
    {snapshot.platform === 'windows'
      ? <WindowsApp snapshot={snapshot} view={view} open={open} perform={perform} error={error} clearError={clearError} />
      : <MacApp snapshot={snapshot} view={view} open={open} perform={perform} error={error} clearError={clearError} />}
  </>
}

createRoot(document.getElementById('root')!).render(<React.StrictMode><RenderBoundary><App /></RenderBoundary></React.StrictMode>)
