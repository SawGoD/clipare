import type { MockBackend } from './mockBackend'
import type { Scenario } from './fixtures'
import type { Platform, Snapshot } from '../domain/types'

const scenarios: Scenario[] = ['default', 'empty', 'offline', 'error', 'pairing', 'update', 'secret']

export function PreviewBar({ backend, snapshot }: { backend: MockBackend; snapshot: Snapshot }) {
  const selectPlatform = (platform: Platform) => {
    const url = new URL(location.href)
    url.searchParams.set('platform', platform)
    location.assign(url.href)
  }
  return <aside className="preview-bar" aria-label="Параметры предпросмотра">
    <span className="preview-title">Предпросмотр</span>
    <label>Платформа <select aria-label="Платформа предпросмотра" value={snapshot.platform} onChange={event => selectPlatform(event.target.value as Platform)}>
      <option value="windows">Windows 11</option><option value="macos">macOS</option>
    </select></label>
    <label>Сценарий <select aria-label="Сценарий предпросмотра" defaultValue={new URLSearchParams(location.search).get('scenario') || 'default'} onChange={event => backend.setScenario(event.target.value as Scenario)}>
      {scenarios.map(scenario => <option key={scenario} value={scenario}>{scenario}</option>)}
    </select></label>
    <label>Тема <select aria-label="Тема предпросмотра" value={snapshot.theme} onChange={event => backend.setTheme(event.target.value as 'light' | 'dark')}>
      <option value="light">Светлая</option><option value="dark">Тёмная</option>
    </select></label>
    <label>Акцент <input aria-label="Цвет акцента" type="color" value={snapshot.accent} onChange={event => backend.setAccent(event.target.value)} /></label>
  </aside>
}
