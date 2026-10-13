export type Platform = 'windows' | 'macos'
export type ThemeChoice = 'system' | 'light' | 'dark'
export type SyncState = 'active' | 'disabled' | 'degraded'
export type View = 'home' | 'discovery' | 'pairing' | 'history' | 'updates'
export type UpdateChannel = 'stable' | 'beta'

export interface Device {
  id: string
  name: string
  platform: 'windows' | 'macos' | 'unknown'
  online: boolean
  address?: string
}

export interface HistoryEntry {
  id: string
  group: string
  source: string
  time: string
  visibleText: string
  sensitive: boolean
  revealed: boolean
  expanded: boolean
  truncated: boolean
  url?: string
}

export interface DiscoveredDevice {
  id: string
  name: string
  address: string
  platform: Device['platform']
}

export interface PairingState {
  direction: 'incoming' | 'outgoing'
  deviceName: string
  code: string
  awaitingApproval: boolean
  needsLocalConfirmation: boolean
}

export type UpdateState =
  | { kind: 'idle' }
  | { kind: 'checking' }
  | { kind: 'current' }
  | { kind: 'available'; version: string }
  | { kind: 'downloading'; version: string; progress?: number }
  | { kind: 'installing'; version: string }
  | { kind: 'ready'; version: string }
  | { kind: 'error'; message: string; retry?: 'check' | 'install' }

export interface Settings {
  autostart: boolean
  autoCheckUpdates: boolean
  updateChannel: UpdateChannel
  historyEnabled: boolean
  historyRetention: '30m' | '1h' | '6h' | '12h' | '24h' | '48h' | '72h'
  historyLimit: number
}

export interface AdvancedSettings {
  deviceId: string
  listenAddress: string
  addresses: string[]
  port: number
  publicKeyFingerprint: string
  legacyKeyPresent: boolean
  legacyConnectionCode?: string
  manualPeer?: { name: string; id: string; address: string; port: number }
}

export interface Snapshot {
  platform: Platform
  theme: 'light' | 'dark'
  accent: string
  version: string
  deviceName: string
  sync: { state: SyncState; message: string; reason?: string }
  devices: Device[]
  history: HistoryEntry[]
  discovered: DiscoveredDevice[]
  discoveryMessage?: string
  pairing?: PairingState
  update: UpdateState
  settings: Settings
  advanced: AdvancedSettings
  busy: boolean
  notice?: string
}

export type Command =
  | { type: 'sync.toggle' }
  | { type: 'device.rename'; name: string }
  | { type: 'device.remove'; id: string }
  | { type: 'device.discover' }
  | { type: 'device.connect'; id: string }
  | { type: 'device.importCode'; code: string }
  | { type: 'pair.approve' | 'pair.reject' | 'pair.cancel' | 'pair.confirmLocal' }
  | { type: 'history.open' | 'history.clear' }
  | { type: 'history.copy' | 'history.delete' | 'history.reveal' | 'history.expand'; id: string }
  | { type: 'history.openLink'; id: string }
  | { type: 'settings.autostart' | 'settings.autoCheckUpdates' | 'settings.historyEnabled'; value: boolean }
  | { type: 'settings.updateChannel'; value: UpdateChannel }
  | { type: 'settings.historyRetention'; value: Settings['historyRetention'] }
  | { type: 'settings.historyLimit'; value: number }
  | { type: 'settings.advanced'; value: Partial<AdvancedSettings> & { deviceName?: string } }
  | { type: 'settings.generateLegacyKey' | 'settings.copyLegacyCode' }
  | { type: 'settings.manualPeer'; value: NonNullable<AdvancedSettings['manualPeer']> }
  | { type: 'update.check' | 'update.install' | 'update.dismiss' }
  | { type: 'view.open'; view: View }
  | { type: 'notice.dismiss' }

export interface Backend {
  snapshot(): Promise<Snapshot>
  perform(command: Command): Promise<void>
  subscribe(listener: (snapshot: Snapshot) => void): () => void
}
