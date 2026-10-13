import type { Backend, Command, Platform, Snapshot } from '../domain/types'
import { fixture, type Scenario } from './fixtures'

const previewSecrets: Record<string, string> = { h2: 'clipare-preview-secret-value' }

export class MockBackend implements Backend {
  private state: Snapshot
  private listeners = new Set<(snapshot: Snapshot) => void>()

  constructor(platform: Platform, scenario: Scenario, theme: 'light' | 'dark') {
    this.state = fixture(platform, scenario, theme)
  }

  async snapshot(): Promise<Snapshot> { return structuredClone(this.state) }

  subscribe(listener: (snapshot: Snapshot) => void): () => void {
    this.listeners.add(listener)
    return () => { this.listeners.delete(listener) }
  }

  private publish(): void {
    const copy = structuredClone(this.state)
    this.listeners.forEach(listener => listener(copy))
  }

  setScenario(scenario: Scenario): void {
    const { platform, theme, accent } = this.state
    this.state = fixture(platform, scenario, theme)
    this.state.accent = accent
    this.publish()
  }

  setTheme(theme: 'light' | 'dark'): void {
    this.state.theme = theme
    this.publish()
  }

  setAccent(accent: string): void {
    this.state.accent = accent
    this.publish()
  }

  async perform(command: Command): Promise<void> {
    const s = this.state
    switch (command.type) {
      case 'sync.toggle':
        s.sync = s.sync.state === 'disabled' ? { state: 'active', message: 'Синхронизация включена' } : { state: 'disabled', message: 'Синхронизация приостановлена' }
        break
      case 'device.rename': s.deviceName = command.name.trim() || s.deviceName; break
      case 'device.remove': s.devices = s.devices.filter(device => device.id !== command.id); break
      case 'device.discover': s.discoveryMessage = s.discovered.length ? undefined : 'Устройства Clipare не найдены'; break
      case 'device.connect': {
        const target = s.discovered.find(device => device.id === command.id)
        if (!target) throw new Error('Устройство больше недоступно')
        s.pairing = { direction: 'outgoing', deviceName: target.name, code: '482 731', awaitingApproval: true, needsLocalConfirmation: s.devices.length > 0 }
        break
      }
      case 'device.importCode':
        if (!command.code.startsWith('clipare1:')) throw new Error('Некорректный код подключения')
        s.notice = 'Код принят в режиме предпросмотра'; break
      case 'pair.approve':
      case 'pair.confirmLocal': {
        const name = s.pairing?.deviceName
        if (name && !s.devices.some(device => device.name === name)) s.devices.push({ id: 'paired-' + name, name, platform: 'unknown', online: true })
        s.pairing = undefined; s.notice = 'Устройство подключено'; break
      }
      case 'pair.reject': case 'pair.cancel': s.pairing = undefined; s.notice = 'Подключение отменено'; break
      case 'history.open': s.history = s.history.map(entry => ({ ...entry, revealed: false, visibleText: entry.sensitive ? '•••• •••• •••• ••••' : entry.visibleText })); break
      case 'history.clear': s.history = []; break
      case 'history.delete': s.history = s.history.filter(entry => entry.id !== command.id); break
      case 'history.copy': s.notice = 'Запись скопирована (предпросмотр)'; break
      case 'history.openLink': s.notice = 'Ссылка открыта (предпросмотр)'; break
      case 'history.reveal':
        s.history = s.history.map(entry => entry.id === command.id ? { ...entry, revealed: !entry.revealed, visibleText: entry.revealed ? '•••• •••• •••• ••••' : previewSecrets[entry.id] || entry.visibleText } : entry)
        break
      case 'history.expand': s.history = s.history.map(entry => entry.id === command.id ? { ...entry, expanded: !entry.expanded } : entry); break
      case 'settings.autostart': s.settings.autostart = command.value; break
      case 'settings.autoCheckUpdates': s.settings.autoCheckUpdates = command.value; break
      case 'settings.historyEnabled': s.settings.historyEnabled = command.value; break
      case 'settings.updateChannel': s.settings.updateChannel = command.value; break
      case 'settings.historyRetention': s.settings.historyRetention = command.value; break
      case 'settings.historyLimit': s.settings.historyLimit = command.value; break
      case 'settings.advanced':
        s.advanced = { ...s.advanced, ...command.value }
        if (command.value.deviceName) s.deviceName = command.value.deviceName
        break
      case 'settings.generateLegacyKey': s.advanced.legacyKeyPresent = true; s.notice = 'Legacy key создан (предпросмотр)'; break
      case 'settings.copyLegacyCode': s.notice = 'Код скопирован (предпросмотр)'; break
      case 'settings.manualPeer': s.devices.push({ id: command.value.id, name: command.value.name, platform: 'unknown', online: false, address: command.value.address }); break
      case 'update.check': s.update = s.update.kind === 'available' ? s.update : { kind: 'current' }; break
      case 'update.install': if (s.update.kind === 'available') s.update = { kind: 'downloading', version: s.update.version, progress: 42 }; break
      case 'update.dismiss': s.update = { kind: 'idle' }; break
      case 'notice.dismiss': s.notice = undefined; break
      case 'view.open': if (command.view === 'history') await this.perform({ type: 'history.open' }); break
    }
    this.publish()
  }
}
