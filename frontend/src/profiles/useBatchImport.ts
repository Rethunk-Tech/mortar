import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { useState } from 'react'
import { ImportAs } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/packsvc/service.ts'
import { showImported } from '../share/useImportFlow.ts'
import { reportUnexpected, toastError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

// The profiles ticked in the import wizard, and the import that turns each into a new profile named after it, read the
// way a single pick would be.
export function useBatchImport(game: string, onDone: () => void) {
  const { t } = useLingui()
  const [chosen, setChosen] = useState<string[]>([])
  const [busy, setBusy] = useState(false)
  const importChosen = async () => {
    setBusy(true)
    let done = 0
    let first = ''
    for (const path of chosen) {
      try {
        const res = await ImportAs({ path, text: '' }, game, '')
        first ||= res.profile
        done += 1
      } catch (error) {
        toastError(t`Could not import`, error)
      }
    }
    setBusy(false)
    if (done > 0) {
      useToasts.getState().push({
        kind: 'success',
        title: plural(done, { one: 'Imported # profile', other: 'Imported # profiles' }),
      })
    }
    onDone()
    // Like a single import, the batch leaves setup for the game, on the first profile it made.
    if (first !== '') {
      showImported(game, false, first).catch(reportUnexpected)
    }
  }
  return { chosen, setChosen, busy, importChosen }
}
