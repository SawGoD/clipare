import { FluentProvider, webDarkTheme, webLightTheme } from '@fluentui/react-components'
import type { PlatformProps } from '../shared'
import { Action } from '../controls'
import { AdditionalSection, AdvancedSection, DeviceSection, Footer, HistorySection, UpdateSection } from '../sections'
import './windows.css'

export default function WindowsApp(props: PlatformProps) {
  const { snapshot: s, view, open, perform, error, clearError } = props
  const base = s.theme === 'dark' ? webDarkTheme : webLightTheme
  const theme = { ...base, colorBrandBackground: s.accent, colorBrandBackgroundHover: s.accent, colorBrandBackgroundPressed: s.accent, colorCompoundBrandBackground: s.accent }
  return <FluentProvider theme={theme} className={`clipare-app win-app ${s.theme}`} style={{ '--accent': s.accent } as React.CSSProperties}>
    <div className="win-shell"><header className="app-header"><h1>Clipare</h1><div className="header-actions">
      <Action platform="windows" onClick={() => void perform({ type: 'sync.toggle' })}>{s.sync.state === 'disabled' ? 'Возобновить' : 'Пауза'}</Action>
      <Action platform="windows" kind="subtle" icon="settings" label="Расширенные параметры" onClick={() => document.getElementById('advanced-settings')?.scrollIntoView({ behavior: 'smooth', block: 'center' })} />
    </div></header>
      {error && <div role="alert" className="inline-error">{error}<Action platform="windows" kind="subtle" icon="close" label="Закрыть сообщение" onClick={clearError} /></div>}
      {s.notice && <div role="status" className="notice-banner">{s.notice}<Action platform="windows" kind="subtle" icon="close" label="Закрыть сообщение" onClick={() => void perform({ type: 'notice.dismiss' })} /></div>}
      <section className="surface self-surface" aria-label="Это устройство"><h2>Это устройство</h2>
        <div className="self-name">{s.deviceName}</div><div className={`sync-status ${s.sync.state}`}><i className="status-dot" /><span>{s.sync.message}</span>{s.sync.reason && <small>{s.sync.reason}</small>}</div>
      </section>
      {view === 'history' ? <HistorySection {...props} platform="windows" /> : view === 'updates' ? <UpdateSection {...props} platform="windows" /> : <DeviceSection {...props} platform="windows" />}
      <AdditionalSection snapshot={s} platform="windows" perform={perform} />
      <section id="advanced-settings"><AdvancedSection snapshot={s} platform="windows" perform={perform} /></section>
      <Footer snapshot={s} platform="windows" open={open} perform={perform} />
      <div className="sr-only" aria-live="polite">{s.sync.message}. {s.devices.filter(d => d.online).length} устройств в сети.</div>
    </div>
  </FluentProvider>
}
