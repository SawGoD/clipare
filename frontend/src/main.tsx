import React, { useEffect, useMemo, useRef } from 'react'
import { createRoot } from 'react-dom/client'
import { useClipare } from './domain/useClipare'
import { WailsBackend } from './domain/wailsBackend'
import { MockBackend } from './preview/mockBackend'
import { PreviewBar } from './preview/PreviewBar'
import type { Platform, View } from './domain/types'
import type { Scenario } from './preview/fixtures'
import WindowsApp from './platform/windows/WindowsApp'
import MacApp from './platform/macos/MacApp'
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
  const { snapshot, error, clearError, perform } = useClipare(backend)
  const [view, setView] = React.useState<View>('home')
  const pairWasShown = useRef(false)
  const previousUpdateKind = useRef(snapshot?.update.kind)

  useEffect(() => {
    if (snapshot?.pairing) {
      pairWasShown.current = true
      if (view !== 'pairing') setView('pairing')
    } else if (pairWasShown.current) {
      pairWasShown.current = false
      if (view === 'pairing') setView('home')
    }
  }, [snapshot?.pairing, view])
  useEffect(() => {
    if (snapshot?.update.kind === 'available' && previousUpdateKind.current !== 'available') setView('updates')
    previousUpdateKind.current = snapshot?.update.kind
  }, [snapshot?.update.kind])

  if (!snapshot) return <main className="loading" role="status">{error || 'Открываем Clipare…'}</main>
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

createRoot(document.getElementById('root')!).render(<React.StrictMode><App /></React.StrictMode>)
