import { useCallback, useEffect, useState } from 'react'
import type { Backend, Command, Snapshot } from './types'

export function useClipare(backend: Backend) {
  const [snapshot, setSnapshot] = useState<Snapshot | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let alive = true
    const off = backend.subscribe(next => { if (alive) setSnapshot(next) })
    backend.snapshot().then(next => { if (alive) setSnapshot(next) }).catch(() => {
      if (alive) setError('Не удалось загрузить состояние Clipare')
    })
    return () => { alive = false; off() }
  }, [backend])

  const perform = useCallback(async (command: Command): Promise<boolean> => {
    try {
      setError(null)
      await backend.perform(command)
      return true
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Не удалось выполнить действие')
      return false
    }
  }, [backend])

  return { snapshot, error, clearError: () => setError(null), perform }
}
