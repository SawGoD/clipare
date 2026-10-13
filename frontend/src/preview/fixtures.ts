import type { Platform, Snapshot } from '../domain/types'

export type Scenario = 'default' | 'empty' | 'offline' | 'error' | 'pairing' | 'update' | 'secret'

export function fixture(platform: Platform, scenario: Scenario = 'default', theme: 'light' | 'dark' = 'light'): Snapshot {
  const devices: Snapshot['devices'] = [
    { id: 'macbook', name: 'MacBook Air', platform: 'macos', online: true, address: '100.100.100.201' },
    { id: 'workstation', name: 'Workstation Development', platform: 'windows', online: false, address: '100.100.100.203' },
  ]
  const history: Snapshot['history'] = [
    { id: 'h1', group: 'Сегодня', source: 'MacBook Air', time: '10:45', visibleText: 'docker compose ps', sensitive: false, revealed: false, expanded: false, truncated: false },
    { id: 'h2', group: 'Сегодня', source: 'Этот компьютер', time: '10:42', visibleText: '•••• •••• •••• ••••', sensitive: true, revealed: false, expanded: false, truncated: false },
    { id: 'h3', group: 'Вчера', source: 'MacBook Air', time: '21:32', visibleText: 'https://github.com/SawGoD/clipare — заметка о выпуске и настройке устройств через NetBird', sensitive: false, revealed: false, expanded: false, truncated: true, url: 'https://github.com/SawGoD/clipare' },
  ]
  return {
    platform, theme, accent: platform === 'windows' ? '#0f6cbd' : '#8b5db6', version: '0.6.0-beta.4',
    deviceName: platform === 'windows' ? 'Desktop-PC' : 'MacBook Air',
    sync: scenario === 'error' ? { state: 'degraded', message: 'Синхронизация недоступна', reason: 'Ожидание NetBird' } : { state: 'active', message: 'Синхронизация включена' },
    devices: scenario === 'empty' ? [] : scenario === 'offline' ? devices.map(device => ({ ...device, online: false })) : devices,
    history, discovered: [
      { id: 'desktop', name: 'Desktop-PC', address: '100.100.100.200', platform: 'windows' },
      { id: 'laptop', name: 'Laptop', address: '100.100.100.204', platform: 'unknown' },
    ],
    discoveryMessage: scenario === 'error' ? 'NetBird не запущен. Проверьте подключение и повторите поиск.' : undefined,
    pairing: scenario === 'pairing' ? { direction: 'incoming', deviceName: 'Desktop-PC', code: '482 731', awaitingApproval: true, needsLocalConfirmation: false } : undefined,
    update: scenario === 'update' ? { kind: 'available', version: '0.6.0-beta.5' } : { kind: 'idle' },
    settings: { autostart: true, autoCheckUpdates: true, updateChannel: 'beta', historyEnabled: true, historyRetention: '24h', historyLimit: 10 },
    advanced: { deviceId: 'eccc9cad83071fed', listenAddress: '100.100.100.201', addresses: ['100.100.100.201', '127.0.0.1'], port: 45873, publicKeyFingerprint: '424f6f1eeb8cf43ee0021e6caffffc3b', legacyKeyPresent: false },
    busy: false,
  }
}
