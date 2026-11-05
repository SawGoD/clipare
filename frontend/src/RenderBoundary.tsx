import { Component, type ErrorInfo, type ReactNode } from 'react'

interface Props { children: ReactNode }
interface State { failed: boolean }

export class RenderBoundary extends Component<Props, State> {
  state: State = { failed: false }

  static getDerivedStateFromError(): State { return { failed: true } }

  componentDidCatch(_error: Error, info: ErrorInfo): void {
    // Component names help diagnose a broken view without logging clipboard text.
    console.error('Clipare UI rendering failed', info.componentStack)
  }

  render(): ReactNode {
    if (this.state.failed) return <main className="recovery-screen" role="alert">
      <div className="recovery-card"><h1>Не удалось показать Clipare</h1>
        <p>Перезагрузите интерфейс. Синхронизация продолжит работать в фоне.</p>
        <button type="button" onClick={() => window.location.reload()}>Перезагрузить интерфейс</button>
      </div>
    </main>
    return this.props.children
  }
}
