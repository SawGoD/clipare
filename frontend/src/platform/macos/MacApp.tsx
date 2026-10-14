import { useState } from 'react'
import type { PlatformProps } from '../shared'
import { Action } from '../controls'
import { AdditionalSection, AdvancedSection, DeviceSection, Footer, HistorySection, UpdateSection } from '../sections'
import './macos.css'

export default function MacApp(props: PlatformProps) {
  const { snapshot: s, view, open, perform, error, clearError } = props
  const [advancedExpanded, setAdvancedExpanded] = useState(false)
  const showAdvanced = () => {
    setAdvancedExpanded(value => !value)
    if (!advancedExpanded) requestAnimationFrame(() => document.getElementById('advanced-settings')?.scrollIntoView({ behavior: 'smooth', block: 'nearest' }))
  }
  return <div className={`clipare-app mac-app ${s.theme} ${import.meta.env.DEV ? '' : 'native-glass'}`} style={{ '--accent': s.accent } as React.CSSProperties}>
    <div className="mac-shell"><header className="app-header"><div className="brand"><h1>Clipare</h1></div>
      <div className="header-actions"><Action platform="macos" onClick={() => void perform({ type: 'sync.toggle' })}>{s.sync.state === 'disabled' ? 'Возобновить' : 'Пауза'}</Action>
        <Action platform="macos" kind="subtle" icon="settings" label="Расширенные параметры" expanded={advancedExpanded} onClick={showAdvanced} /></div></header>
      {error && <div role="alert" className="inline-error">{error}<Action platform="macos" kind="subtle" icon="close" label="Закрыть сообщение" onClick={clearError} /></div>}
      {s.notice && <div role="status" className="notice-banner">{s.notice}<Action platform="macos" kind="subtle" icon="close" label="Закрыть сообщение" onClick={() => void perform({ type: 'notice.dismiss' })} /></div>}
      <section className="surface self-surface" aria-label="Это устройство"><div className="mac-self-top"><div><small>ЭТО УСТРОЙСТВО</small><h2>{s.deviceName}</h2></div><div className={`sync-status ${s.sync.state}`}><i className="status-dot" /><span>{s.sync.message}</span></div></div>
        {s.sync.reason && <p className="status-reason">{s.sync.reason}</p>}</section>
      {view === 'history' ? <HistorySection {...props} platform="macos" /> : view === 'updates' ? <UpdateSection {...props} platform="macos" /> : <DeviceSection {...props} platform="macos" />}
      <div className="mac-preferences"><AdditionalSection snapshot={s} platform="macos" perform={perform} />
        {advancedExpanded && <div id="advanced-settings"><AdvancedSection snapshot={s} platform="macos" perform={perform} /></div>}</div>
      <Footer snapshot={s} platform="macos" open={open} perform={perform} />
      <div className="sr-only" aria-live="polite">{s.sync.message}. {s.devices.filter(d => d.online).length} устройств в сети.</div>
    </div>
  </div>
}
