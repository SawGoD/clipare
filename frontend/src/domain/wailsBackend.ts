import { Call, Events } from '@wailsio/runtime'
import type { Backend, Command, Snapshot } from './types'

// Only these two service methods are callable by the GUI. The Go service
// validates the discriminated command and owns every privileged operation.
const service = 'ui.ReactService'

export class WailsBackend implements Backend {
  async snapshot(): Promise<Snapshot> {
    return await Call.ByName(`${service}.Snapshot`) as Snapshot
  }

  async perform(command: Command): Promise<void> {
    await Call.ByName(`${service}.Perform`, command)
  }

  subscribe(listener: (snapshot: Snapshot) => void): () => void {
    return Events.On('clipare:state', event => listener(event.data as Snapshot))
  }
}
