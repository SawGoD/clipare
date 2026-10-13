import type { Command, Snapshot, View } from '../domain/types'

export interface PlatformProps {
  snapshot: Snapshot
  view: View
  open(view: View): void
  perform(command: Command): Promise<boolean>
  error: string | null
  clearError(): void
}

export function versionLabel(version: string): string {
  return version.includes('-beta.') ? `${version} (бета)` : version
}
