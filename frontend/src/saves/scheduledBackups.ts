import { msg } from '@lingui/core/macro'
import { Events } from '@wailsio/runtime'
import { useEffect, useState } from 'react'
import {
  LastScheduledBackup,
  OpenBackupsFolder,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savessvc/service.ts'
import { i18n } from '../i18n/index.ts'
import { reportUnexpected, toastError } from '../toasts/report.ts'

const EVENT = 'saves:scheduled'
const HOUR_MS = 3_600_000

interface ScheduledRun {
  at: number
  saved: number
  unchanged: number
  failed: number
  error: string
}

/** A scheduled pass that failed to back up a save is shown at once; successful passes stay quiet. */
export function initScheduledBackups() {
  Events.On(EVENT, (event) => {
    const run = event.data as ScheduledRun
    if (run.failed > 0 || run.error !== '') {
      toastError(i18n._(msg`A scheduled save backup failed`), new Error(run.error), {
        action: {
          label: i18n._(msg`Open backups folder`),
          run: () => OpenBackupsFolder().catch(reportUnexpected),
        },
      })
    }
  })
}

/** Whole hours until the next scheduled backup is due, at least 1 while one is still ahead; 0 when it is due now. */
export function hoursUntilNext(last: number, hours: number, now = Date.now()): number {
  const due = last + hours * HOUR_MS
  return due > now ? Math.max(1, Math.ceil((due - now) / HOUR_MS)) : 0
}

/** When the scheduled backup last ran (Unix ms, 0 for never), reloaded after each pass. */
export function useLastScheduled() {
  const [last, setLast] = useState(0)
  useEffect(() => {
    const load = () => LastScheduledBackup().then(setLast).catch(reportUnexpected)
    load()
    return Events.On(EVENT, load)
  }, [])
  return last
}
