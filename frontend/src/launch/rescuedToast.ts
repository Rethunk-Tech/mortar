import { msg } from '@lingui/core/macro'
import { Events } from '@wailsio/runtime'
import { i18n } from '../i18n/index.ts'
import { useToasts } from '../toasts/store.ts'

const EVENT = 'launch:rescued'
const DISCARD = 'launch:discard-rescued'

function showRescued(notice: { paths?: string[] } | null) {
  const paths = (notice?.paths ?? []).filter((p) => p !== '')
  if (paths.length === 0) {
    return
  }
  useToasts.getState().push({
    kind: 'info',
    title: i18n._(msg`Edits were restored`),
    action: {
      label: i18n._(msg`Discard`),
      run: () => Events.Emit(DISCARD, { paths }),
    },
  })
}

function initRescuedToast() {
  Events.On(EVENT, (event) => showRescued(event.data as { paths?: string[] }))
}

export { initRescuedToast, showRescued }
