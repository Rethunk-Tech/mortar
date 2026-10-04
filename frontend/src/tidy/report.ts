import { msg, plural } from '@lingui/core/macro'
import { Events } from '@wailsio/runtime'
import { Take } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/tidy/service.ts'
import { i18n } from '../i18n/index.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

async function show() {
  const report = await Take()
  if (report.total <= 0) {
    return
  }
  const lines = (report.entries ?? []).map((entry) => {
    const names = (entry.names ?? []).join(', ')
    const what = entry.count > 1 ? `${entry.what} (${entry.count})` : entry.what
    return names === '' ? `${what}: ${entry.where}` : `${what}: ${entry.where}: ${names}`
  })
  useToasts.getState().push({
    kind: 'info',
    title: i18n._(
      msg`Mortar tidied up ${plural(report.total, { one: '# thing', other: '# things' })}`,
    ),
    detail: lines.join('\n'),
  })
}

let listening = false

// Startup repairs finish about when the window loads: ask once now, and again if they finish later.
export function initTidyReport() {
  if (listening || typeof Events.On !== 'function') {
    return
  }
  listening = true
  Events.On('tidy:report', () => {
    show().catch(reportUnexpected)
  })
  show().catch(reportUnexpected)
}
