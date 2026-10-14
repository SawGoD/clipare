import type { ReactNode } from 'react'
import { Button, Checkbox, Dropdown, Input, Option, Switch } from '@fluentui/react-components'
import {
  Add20Regular, ArrowClockwise20Regular, ArrowLeft20Regular, Checkmark20Regular,
  ChevronDown20Regular, Copy20Regular, Delete20Regular, Dismiss20Regular,
  Eye20Regular, EyeOff20Regular, History20Regular, Laptop20Regular,
  Link20Regular, MoreHorizontal20Regular, Settings20Regular, Desktop20Regular,
} from '@fluentui/react-icons'
import type { Platform } from '../domain/types'

export type IconName = 'add' | 'refresh' | 'back' | 'check' | 'down' | 'copy' | 'delete' | 'close' | 'eye' | 'eyeOff' | 'history' | 'laptop' | 'link' | 'more' | 'settings' | 'desktop' | 'platformWindows' | 'platformMac' | 'device'

const windowsMark = <svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor" aria-hidden="true"><path d="M2.5 4.8 11 3.6v7.7H2.5Zm9.8-1.4 9.2-1.3v9.2h-9.2ZM2.5 12.7H11v7.7l-8.5-1.2Zm9.8 0h9.2v9.2l-9.2-1.3Z" /></svg>
const genericDevice = <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true"><rect x="3" y="4" width="18" height="13" rx="2" /><path d="M8 20h8M12 17v3" /><circle cx="18" cy="14" r=".7" fill="currentColor" /></svg>

const winIcons: Record<IconName, ReactNode> = {
  add: <Add20Regular />, refresh: <ArrowClockwise20Regular />, back: <ArrowLeft20Regular />,
  check: <Checkmark20Regular />, down: <ChevronDown20Regular />, copy: <Copy20Regular />,
  delete: <Delete20Regular />, close: <Dismiss20Regular />, eye: <Eye20Regular />,
  eyeOff: <EyeOff20Regular />, history: <History20Regular />, laptop: <Laptop20Regular />,
  link: <Link20Regular />, more: <MoreHorizontal20Regular />, settings: <Settings20Regular />,
  desktop: <Desktop20Regular />,
  platformWindows: windowsMark, platformMac: <Laptop20Regular />, device: genericDevice,
}

const paths: Record<IconName, ReactNode> = {
  add: <path d="M12 5v14M5 12h14" />,
  refresh: <><path d="M20 11a8 8 0 1 0-2.3 6.7" /><path d="M20 4v7h-7" /></>,
  back: <path d="m14 5-7 7 7 7M7 12h13" />,
  check: <path d="m4 12 5 5L20 6" />,
  down: <path d="m5 9 7 7 7-7" />,
  copy: <><rect x="8" y="8" width="12" height="12" rx="2" /><path d="M16 8V6a2 2 0 0 0-2-2H6a2 2 0 0 0-2 2v8a2 2 0 0 0 2 2h2" /></>,
  delete: <><path d="M4 7h16M10 4h4M6 7l1 13h10l1-13M10 11v6M14 11v6" /></>,
  close: <path d="M5 5l14 14M19 5 5 19" />,
  eye: <><path d="M2 12s3.5-6 10-6 10 6 10 6-3.5 6-10 6-10-6-10-6Z" /><circle cx="12" cy="12" r="2.5" /></>,
  eyeOff: <><path d="M3 3l18 18M9 6.5A11 11 0 0 1 12 6c6.5 0 10 6 10 6a16 16 0 0 1-3.1 3.5M6 8.2A16 16 0 0 0 2 12s3.5 6 10 6c1.1 0 2.1-.2 3-.5" /></>,
  history: <><path d="M3 7v5h5M4 12a8 8 0 1 1 2.3 5.7M12 8v5l3 2" /></>,
  laptop: <><rect x="4" y="4" width="16" height="12" rx="1" /><path d="M2 19h20" /></>,
  link: <><path d="M10 13a5 5 0 0 0 7 0l3-3a5 5 0 0 0-7-7l-2 2M14 11a5 5 0 0 0-7 0l-3 3a5 5 0 0 0 7 7l2-2" /></>,
  more: <><circle cx="5" cy="12" r="1" /><circle cx="12" cy="12" r="1" /><circle cx="19" cy="12" r="1" /></>,
  settings: <><circle cx="12" cy="12" r="3" /><path d="M9.8 3.4 10 2h4l.2 1.4 1.8.8 1.2-.7 2.8 2.8-.7 1.2.8 1.8L22 10v4l-1.4.2-.8 1.8.7 1.2-2.8 2.8-1.2-.7-1.8.8L14 22h-4l-.2-1.4-1.8-.8-1.2.7L4 17.7l.7-1.2-.8-1.8L2 14v-4l1.4-.2.8-1.8-.7-1.2L6.3 4l1.2.7 2.3-1.3Z" /></>,
  desktop: <><rect x="3" y="4" width="18" height="13" rx="1.5" /><path d="M12 17v3M8 20h8" /></>,
  platformWindows: <path d="M2.5 4.8 11 3.6v7.7H2.5Zm9.8-1.4 9.2-1.3v9.2h-9.2ZM2.5 12.7H11v7.7l-8.5-1.2Zm9.8 0h9.2v9.2l-9.2-1.3Z" fill="currentColor" stroke="none" />,
  platformMac: <><rect x="4" y="4" width="16" height="12" rx="1.5" /><path d="M2 19h20" /></>,
  device: <><rect x="3" y="4" width="18" height="13" rx="2" /><path d="M8 20h8M12 17v3" /><circle cx="18" cy="14" r=".7" fill="currentColor" /></>,
}

export function Glyph({ platform, name }: { platform: Platform; name: IconName }) {
  if (platform === 'windows') return <>{winIcons[name]}</>
  return <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">{paths[name]}</svg>
}

export function Action({ platform, children, icon, label, kind = 'secondary', disabled, onClick, className = '', expanded }: {
  platform: Platform; children?: ReactNode; icon?: IconName; label?: string; kind?: 'primary' | 'secondary' | 'subtle' | 'danger'; disabled?: boolean; onClick?: () => void; className?: string; expanded?: boolean
}) {
  const accessible = label || (typeof children === 'string' ? children : undefined)
  if (platform === 'windows') return <Button title={label} aria-label={accessible} aria-expanded={expanded} className={`action ${kind} ${!children ? 'icon-only' : ''} ${className}`} appearance={kind === 'primary' ? 'primary' : kind === 'subtle' || kind === 'danger' ? 'subtle' : 'secondary'} icon={icon && <Glyph platform={platform} name={icon} />} disabled={disabled} onClick={onClick}>{children}</Button>
  return <button type="button" title={label} aria-label={accessible} aria-expanded={expanded} className={`action ${kind} ${className}`} disabled={disabled} onClick={onClick}>{icon && <Glyph platform={platform} name={icon} />}{children && <span>{children}</span>}</button>
}

export function Toggle({ platform, label, checked, onChange }: { platform: Platform; label: string; checked: boolean; onChange(value: boolean): void }) {
  if (platform === 'windows') return <Checkbox label={label} checked={checked} onChange={(_, data) => onChange(Boolean(data.checked))} />
  return <label className="toggle-row"><span>{label}</span><input type="checkbox" role="switch" checked={checked} onChange={event => onChange(event.target.checked)} /><span className="switch-track" aria-hidden="true" /></label>
}

export function Field({ platform, label, value, onChange, disabled, placeholder, type = 'text' }: { platform: Platform; label: string; value: string; onChange(value: string): void; disabled?: boolean; placeholder?: string; type?: string }) {
  return <label className="field"><span>{label}</span>{platform === 'windows'
    ? <Input aria-label={label} type={type === 'number' ? 'text' : type as 'text' | 'password'} inputMode={type === 'number' ? 'numeric' : undefined} value={value} placeholder={placeholder} disabled={disabled} onChange={(_, data) => onChange(data.value)} />
    : <input aria-label={label} type={type} value={value} placeholder={placeholder} disabled={disabled} onChange={event => onChange(event.target.value)} />}</label>
}

export function Choice({ platform, label, value, values, onChange }: { platform: Platform; label: string; value: string; values: { value: string; label: string }[]; onChange(value: string): void }) {
  return <div className="field choice"><span>{label}</span>{platform === 'windows'
    ? <Dropdown inlinePopup aria-label={label} value={values.find(item => item.value === value)?.label || value} selectedOptions={[value]} onOptionSelect={(_, data) => onChange(data.optionValue || value)}>
        {values.map(item => <Option key={item.value} value={item.value} text={item.label}>{item.label}</Option>)}
      </Dropdown>
    : <select aria-label={label} value={value} onChange={event => onChange(event.target.value)}>{values.map(item => <option key={item.value} value={item.value}>{item.label}</option>)}</select>}</div>
}

export function CompactToggle({ platform, checked, onChange, label }: { platform: Platform; checked: boolean; onChange(value: boolean): void; label: string }) {
  if (platform === 'windows') return <Switch aria-label={label} checked={checked} onChange={(_, data) => onChange(data.checked)} />
  return <Toggle platform={platform} label={label} checked={checked} onChange={onChange} />
}
