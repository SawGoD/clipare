import { useEffect, useState } from 'react'
import type { AdvancedSettings, Device, HistoryEntry, Platform } from '../domain/types'
import type { PlatformProps } from './shared'
import { versionLabel } from './shared'
import { Action, Choice, Field, Glyph, Toggle } from './controls'

type SectionProps = PlatformProps & { platform: Platform }

export function DeviceSection({ snapshot, platform, view, open, perform }: SectionProps) {
  const [removing, setRemoving] = useState<string | null>(null)
  const devices = snapshot.devices
  const inFlow = view === 'discovery' || view === 'pairing'
  return <section className="surface devices-surface" aria-label="Устройства">
    <div className="section-heading">
      {inFlow && <Action platform={platform} kind="subtle" icon="back" label="К устройствам" onClick={() => {
        if (view === 'pairing') {
          void perform({ type: snapshot.pairing?.direction === 'incoming' ? 'pair.reject' : 'pair.cancel' })
          if (!snapshot.pairing) open('home')
        }
        else open('home')
      }} />}
      <h2>{view === 'discovery' ? 'Добавить устройство' : view === 'pairing' ? 'Подключение устройства' : 'Устройства'}</h2>
      {!inFlow && <Action platform={platform} kind="subtle" icon="history" label="История буфера обмена" onClick={() => open('history')} />}
    </div>
    {view === 'discovery' ? <DiscoverySection snapshot={snapshot} platform={platform} open={open} perform={perform} />
      : view === 'pairing' ? <PairingSection snapshot={snapshot} platform={platform} open={open} perform={perform} />
      : devices.length === 0 ? <div className="empty-devices">
        <Action platform={platform} kind="primary" icon="add" label="Добавить устройство" className="big-add" onClick={() => { open('discovery'); void perform({ type: 'device.discover' }) }} />
        <span>Добавить устройство</span>
      </div> : <>
        <div className="device-list">{devices.map(device => <DeviceRow key={device.id} device={device} platform={platform} removing={removing === device.id}
          onRemove={() => setRemoving(device.id)} onCancel={() => setRemoving(null)} onConfirm={() => { void perform({ type: 'device.remove', id: device.id }); setRemoving(null) }} />)}</div>
        <Action platform={platform} kind="secondary" icon="add" onClick={() => { open('discovery'); void perform({ type: 'device.discover' }) }}>Добавить устройство</Action>
      </>}
  </section>
}

function DeviceRow({ device, platform, removing, onRemove, onCancel, onConfirm }: {
  device: Device; platform: Platform; removing: boolean; onRemove(): void; onCancel(): void; onConfirm(): void
}) {
  return <div className="device-row" data-online={device.online}>
    <div className="device-platform" aria-label={device.platform === 'unknown' ? 'Устройство' : device.platform === 'windows' ? 'Windows' : 'macOS'}>
      <Glyph platform={platform} name={device.platform === 'unknown' ? 'desktop' : device.platform === 'windows' ? 'desktop' : 'laptop'} />
    </div>
    <div className="device-copy"><strong title={device.name}>{device.name}</strong><small><i className="peer-dot" />{device.online ? 'Подключено' : 'Не в сети'}</small></div>
    {removing ? <div className="row-confirm"><span>Удалить?</span><Action platform={platform} kind="danger" onClick={onConfirm}>Да</Action><Action platform={platform} kind="subtle" onClick={onCancel}>Отмена</Action></div>
      : <Action platform={platform} kind="subtle" icon="delete" label={`Удалить ${device.name}`} className="delete-action" onClick={onRemove} />}
  </div>
}

function DiscoverySection({ snapshot, platform, open, perform }: Pick<SectionProps, 'snapshot' | 'platform' | 'open' | 'perform'>) {
  const [code, setCode] = useState('')
  const devices = snapshot.discovered.filter(device => !snapshot.devices.some(known => known.id === device.id))
  return <div className="flow-content">
    {snapshot.discoveryMessage && <p className="hint">{snapshot.discoveryMessage}</p>}
    {devices.length === 0 ? <div className="flow-empty"><Glyph platform={platform} name="desktop" /><strong>Устройства Clipare не найдены</strong><p>Убедитесь, что NetBird и Clipare запущены на других устройствах.</p></div>
      : <div className="discovered-list">{devices.map(device => <div className="discovered-row" key={device.id}>
        <Glyph platform={platform} name={device.platform === 'macos' ? 'laptop' : 'desktop'} /><div><strong>{device.name}</strong><small>{device.address}</small></div>
        <Action platform={platform} kind="primary" onClick={() => { void perform({ type: 'device.connect', id: device.id }); open('pairing') }}>Подключить</Action>
      </div>)}</div>}
    <div className="flow-actions"><Action platform={platform} kind="subtle" icon="refresh" label="Обновить список устройств" onClick={() => void perform({ type: 'device.discover' })} /></div>
    <div className="manual-fallback"><p>Не нашли устройство? Добавьте его по коду.</p><div className="inline-form"><Field platform={platform} label="Код подключения" value={code} onChange={setCode} placeholder="clipare1:…" /><Action platform={platform} kind="secondary" disabled={!code.trim()} onClick={async () => { if (await perform({ type: 'device.importCode', code: code.trim() })) { setCode(''); open('home') } }}>Добавить по коду</Action></div></div>
  </div>
}

function PairingSection({ snapshot, platform, open, perform }: Pick<SectionProps, 'snapshot' | 'platform' | 'open' | 'perform'>) {
  const pair = snapshot.pairing
  if (!pair) return <div className="flow-empty"><strong>Подключаем устройство…</strong><Action platform={platform} onClick={() => { void perform({ type: 'pair.cancel' }); open('home') }}>Отменить</Action></div>
  return <div className="pair-flow"><p>{pair.direction === 'incoming' ? `${pair.deviceName} хочет подключиться` : `Проверьте код на ${pair.deviceName}`}</p>
    <div className="sas-card"><small>Код проверки</small><strong aria-label={`Код проверки ${pair.code}`}>{pair.code}</strong></div>
    <p className="hint">Сравните код на обоих устройствах. Если коды различаются — отклоните подключение.</p>
    <div className="flow-actions">{pair.direction === 'incoming' ? <>
      <Action platform={platform} kind="primary" onClick={() => void perform({ type: 'pair.approve' })}>Разрешить</Action>
      <Action platform={platform} onClick={() => void perform({ type: 'pair.reject' })}>Отклонить</Action>
    </> : <>
      {pair.needsLocalConfirmation && <Action platform={platform} kind="primary" onClick={() => void perform({ type: 'pair.confirmLocal' })}>Коды совпадают</Action>}
      <span className="hint">Ожидание подтверждения…</span>
      <Action platform={platform} onClick={() => void perform({ type: 'pair.cancel' })}>Отменить</Action>
    </>}</div>
  </div>
}

export function HistorySection({ snapshot, platform, open, perform }: SectionProps) {
  const [confirmClear, setConfirmClear] = useState(false)
  let lastGroup = ''
  return <section className="surface history-surface" aria-label="История">
    <div className="section-heading"><Action platform={platform} kind="subtle" icon="back" label="К устройствам" onClick={() => open('home')} /><h2>История</h2>
      {snapshot.history.length > 0 && <Action platform={platform} kind="subtle" icon="delete" label="Очистить историю" onClick={() => setConfirmClear(true)} />}</div>
    {confirmClear && <div className="inline-error">Очистить всю историю на этом устройстве?<Action platform={platform} kind="danger" onClick={() => { void perform({ type: 'history.clear' }); setConfirmClear(false) }}>Очистить</Action><Action platform={platform} onClick={() => setConfirmClear(false)}>Отмена</Action></div>}
    {snapshot.history.length === 0 ? <div className="flow-empty"><Glyph platform={platform} name="history" /><strong>История пока пуста</strong><p>Скопированный текст появится здесь, когда сохранение истории включено.</p></div>
      : <div className="history-list">{snapshot.history.map(entry => {
        const showGroup = lastGroup !== entry.group
        lastGroup = entry.group
        return <div key={entry.id}>{showGroup && <div className="history-group">{entry.group}</div>}<HistoryRow entry={entry} platform={platform} perform={perform} /></div>
      })}</div>}
  </section>
}

function HistoryRow({ entry, platform, perform }: { entry: HistoryEntry; platform: Platform; perform: SectionProps['perform'] }) {
  return <article className="history-row"><div className="history-row-top"><small>{entry.source} · {entry.time}</small><div className="row-actions">
    {entry.truncated && <Action platform={platform} kind="subtle" icon="down" label={entry.expanded ? 'Свернуть запись' : 'Развернуть запись'} onClick={() => void perform({ type: 'history.expand', id: entry.id })} />}
    {entry.sensitive && <Action platform={platform} kind="subtle" icon={entry.revealed ? 'eyeOff' : 'eye'} label={entry.revealed ? 'Скрыть значение' : 'Показать значение'} onClick={() => void perform({ type: 'history.reveal', id: entry.id })} />}
    {entry.url && <Action platform={platform} kind="subtle" icon="link" label="Открыть ссылку" onClick={() => void perform({ type: 'history.openLink', id: entry.id })} />}
    <Action platform={platform} kind="subtle" icon="copy" label="Копировать запись" onClick={() => void perform({ type: 'history.copy', id: entry.id })} />
    <Action platform={platform} kind="subtle" icon="delete" label="Удалить запись" className="delete-action" onClick={() => void perform({ type: 'history.delete', id: entry.id })} />
  </div></div><p className={entry.expanded ? 'history-text expanded' : 'history-text'}>{entry.visibleText}</p></article>
}

export function UpdateSection({ snapshot, platform, open, perform }: SectionProps) {
  const state = snapshot.update
  return <section className="surface update-surface" aria-label="Обновления"><div className="section-heading">
    <Action platform={platform} kind="subtle" icon="back" label="К устройствам" onClick={() => open('home')} /><h2>Обновления</h2></div>
    <div className="update-content">{state.kind === 'idle' && <><strong>Проверить обновления Clipare</strong><p>Установлена версия {versionLabel(snapshot.version)}</p><Action platform={platform} kind="primary" onClick={() => void perform({ type: 'update.check' })}>Проверить обновления</Action></>}
      {state.kind === 'checking' && <><strong>Проверяем обновления…</strong><progress /></>}
      {state.kind === 'current' && <><strong>Установлена последняя версия Clipare</strong><p>Текущая версия: {versionLabel(snapshot.version)}</p><Action platform={platform} onClick={() => { void perform({ type: 'update.dismiss' }); open('home') }}>Понятно</Action></>}
      {state.kind === 'available' && <><strong>Доступна версия {versionLabel(state.version)}</strong><p>Установлена: {versionLabel(snapshot.version)}</p><div className="flow-actions"><Action platform={platform} kind="primary" onClick={() => void perform({ type: 'update.install' })}>Обновить</Action><Action platform={platform} onClick={() => { void perform({ type: 'update.dismiss' }); open('home') }}>Позже</Action></div></>}
      {state.kind === 'downloading' && <><strong>Загрузка Clipare {versionLabel(state.version)}</strong><progress max={100} value={state.progress} aria-label="Загрузка обновления" /></>}
      {state.kind === 'installing' && <><strong>Обновление готово</strong><p>Clipare перезапустится для установки…</p><progress aria-label="Установка обновления" /></>}
      {state.kind === 'ready' && <><strong>Обновление готово</strong><p>Clipare перезапустится для установки.</p><Action platform={platform} kind="primary" onClick={() => void perform({ type: 'update.install' })}>Установить и перезапустить</Action></>}
      {state.kind === 'error' && <><strong>Не удалось обновить Clipare</strong><p>{state.message}</p><div className="flow-actions"><Action platform={platform} kind="primary" onClick={() => void perform({ type: state.retry === 'install' ? 'update.install' : 'update.check' })}>Повторить</Action><Action platform={platform} onClick={() => open('home')}>Закрыть</Action></div></>}
    </div></section>
}

export function AdditionalSection({ snapshot, platform, perform }: Pick<SectionProps, 'snapshot' | 'platform' | 'perform'>) {
  const [expanded, setExpanded] = useState(false)
  const s = snapshot.settings
  return <section className="disclosure"><button type="button" className="disclosure-trigger" aria-expanded={expanded} onClick={() => setExpanded(!expanded)}><span>Дополнительно</span><Glyph platform={platform} name="down" /></button>
    {expanded && <div className="disclosure-body"><Toggle platform={platform} label="Запускать при входе в систему" checked={s.autostart} onChange={value => void perform({ type: 'settings.autostart', value })} />
      <div className="settings-inline"><Toggle platform={platform} label="Авто-проверка обновлений" checked={s.autoCheckUpdates} onChange={value => void perform({ type: 'settings.autoCheckUpdates', value })} />
        <Choice platform={platform} label="Канал обновлений" value={s.updateChannel} values={[{ value: 'stable', label: 'Стабильные' }, { value: 'beta', label: 'Бета' }]} onChange={value => void perform({ type: 'settings.updateChannel', value: value as 'stable' | 'beta' })} /></div>
      <Toggle platform={platform} label="Сохранять историю" checked={s.historyEnabled} onChange={value => void perform({ type: 'settings.historyEnabled', value })} />
      {s.historyEnabled && <div className="settings-inline"><Choice platform={platform} label="Хранить" value={s.historyRetention} values={[{ value: '30m', label: '30 минут' }, { value: '1h', label: '1 час' }, { value: '6h', label: '6 часов' }, { value: '12h', label: '12 часов' }, { value: '24h', label: '1 день' }, { value: '48h', label: '2 дня' }, { value: '72h', label: '3 дня' }]} onChange={value => void perform({ type: 'settings.historyRetention', value: value as typeof s.historyRetention })} />
        <Field platform={platform} label="Максимум записей" value={String(s.historyLimit)} type="number" onChange={value => { const count = Number(value); if (count > 0) void perform({ type: 'settings.historyLimit', value: count }) }} /></div>}
    </div>}</section>
}

export function AdvancedSection({ snapshot, platform, perform }: Pick<SectionProps, 'snapshot' | 'platform' | 'perform'>) {
  const [expanded, setExpanded] = useState(false)
  const [draft, setDraft] = useState<AdvancedSettings>(snapshot.advanced)
  const [deviceName, setDeviceName] = useState(snapshot.deviceName)
  const [code, setCode] = useState('')
  const [peer, setPeer] = useState({ name: '', id: '', address: '', port: 45873 })
  useEffect(() => { setDraft(snapshot.advanced) }, [snapshot.advanced])
  useEffect(() => { setDeviceName(snapshot.deviceName) }, [snapshot.deviceName])
  const dirty = deviceName.trim() !== snapshot.deviceName || draft.listenAddress !== snapshot.advanced.listenAddress || draft.port !== snapshot.advanced.port
  const change = (field: keyof AdvancedSettings, value: string | number) => setDraft({ ...draft, [field]: value })
  return <section className="disclosure technical"><button type="button" className="disclosure-trigger" aria-expanded={expanded} onClick={() => setExpanded(!expanded)}><span>Расширенные параметры</span><Glyph platform={platform} name="down" /></button>
    {expanded && <div className="disclosure-body"><div className="advanced-grid">
      <Field platform={platform} label="Имя устройства" value={deviceName} onChange={setDeviceName} />
      <Field platform={platform} label="Device ID" value={draft.deviceId} onChange={() => {}} disabled />
      <Choice platform={platform} label="NetBird IP" value={draft.listenAddress} values={draft.addresses.map(address => ({ value: address, label: address }))} onChange={value => change('listenAddress', value)} />
      <Field platform={platform} label="Порт" value={String(draft.port)} type="number" onChange={value => change('port', Number(value))} />
    </div><p className="technical-note">Протокол 1 · Public key SHA-256: {draft.publicKeyFingerprint}</p>
      {dirty && <Action platform={platform} kind="primary" disabled={!deviceName.trim() || draft.port < 1 || draft.port > 65535 || !draft.listenAddress} onClick={() => void perform({ type: 'settings.advanced', value: { deviceName: deviceName.trim(), listenAddress: draft.listenAddress, port: draft.port } })}>Применить</Action>}
      <div className="technical-divider"><span>Legacy и ручное подключение</span><small>Только для старых версий или диагностики</small></div>
      <div className="legacy-row"><span>Общий legacy key: {draft.legacyKeyPresent ? 'создан' : 'не создан'}</span><Action platform={platform} onClick={() => void perform({ type: 'settings.generateLegacyKey' })}>Создать ключ</Action><Action platform={platform} disabled={!draft.legacyKeyPresent} onClick={() => void perform({ type: 'settings.copyLegacyCode' })}>Копировать код</Action></div>
      <div className="inline-form"><Field platform={platform} label="Код подключения" value={code} onChange={setCode} placeholder="clipare1:…" /><Action platform={platform} disabled={!code.trim()} onClick={() => void perform({ type: 'device.importCode', code: code.trim() })}>Добавить по коду</Action></div>
      <div className="advanced-grid manual-grid"><Field platform={platform} label="Имя устройства" value={peer.name} onChange={value => setPeer({ ...peer, name: value })} /><Field platform={platform} label="Device ID другого устройства" value={peer.id} onChange={value => setPeer({ ...peer, id: value })} /><Field platform={platform} label="IP или FQDN" value={peer.address} onChange={value => setPeer({ ...peer, address: value })} /><Field platform={platform} label="Порт устройства" value={String(peer.port)} type="number" onChange={value => setPeer({ ...peer, port: Number(value) })} /></div>
      <Action platform={platform} disabled={!peer.name.trim() || !peer.id.trim() || !peer.address.trim() || peer.port < 1 || peer.port > 65535} onClick={() => void perform({ type: 'settings.manualPeer', value: peer })}>Добавить вручную</Action>
    </div>}</section>
}

export function Footer({ snapshot, platform, open, perform }: Pick<SectionProps, 'snapshot' | 'platform' | 'open' | 'perform'>) {
  return <footer className="app-footer"><span>Версия: {versionLabel(snapshot.version)}</span><Action platform={platform} kind="subtle" icon="refresh" label="Проверить обновления" onClick={() => { open('updates'); void perform({ type: 'update.check' }) }} /></footer>
}
