import { msg } from '@lingui/core/macro'
import { Events } from '@wailsio/runtime'
import { useConsole } from '../console/store.ts'
import { i18n } from '../i18n/index.ts'
import { useToasts } from '../toasts/store.ts'
import { useLaunch } from './store.ts'

export function initLaunch() {
  Events.On('launch:state', (event) => useLaunch.getState().apply(event.data))
  Events.On('launch:line', (event) => useConsole.getState().add(event.data))
  Events.On('launch:backup-warning', (event) => {
    const data = event.data as { error?: string }
    useToasts.getState().push({
      kind: 'warning',
      title: i18n._(msg`Could not back up saves before Play`),
      body: data.error ?? '',
    })
  })
  Events.On('launch:settings-restore-warning', (event) => {
    const data = event.data as { error?: string }
    useToasts.getState().push({
      kind: 'error',
      title: i18n._(msg`Could not restore profile game settings`),
      body: data.error ?? '',
    })
  })
  Events.On('launch:crash', (event) => useLaunch.getState().setCrash(event.data))
}
