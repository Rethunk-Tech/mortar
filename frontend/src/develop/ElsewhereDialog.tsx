import { useLingui } from '@lingui/react/macro'
import { useState } from 'react'
import { Add } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import { useGameName } from '../games/info.ts'
import { ConfirmationDialog } from '../shell/dialog/ConfirmationDialog.tsx'
import { reportError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { useElsewhere } from './elsewhere.ts'

/** Offers to add the folder under the game its project file names instead. */
export function ElsewhereHost() {
  const { t } = useLingui()
  const asked = useElsewhere((s) => s.asked)
  const clear = useElsewhere((s) => s.clear)
  const there = useGameName(asked?.found.game)
  const here = useGameName(asked?.game)
  const [busy, setBusy] = useState(false)
  const add = () => {
    if (!asked) {
      return
    }
    setBusy(true)
    // Its profile is another game's, so the recorded game starts its own development profile.
    Add(asked.found.game, asked.dir, '')
      .then((project) => {
        useToasts
          .getState()
          .push({ kind: 'success', title: t`Added ${{ name: project.name }}`, body: project.dir })
        clear()
      })
      .catch(reportError(t`Could not add the project`))
      .finally(() => setBusy(false))
  }
  return (
    <ConfirmationDialog
      open={asked !== null}
      title={t`This is a ${{ game: there }} project`}
      body={t`${{ dir: asked?.dir ?? '' }} belongs to ${{ game: there }}, not ${{ here }}. Add it under ${{ game: there }} instead?`}
      confirmLabel={t`Add under ${{ game: there }}`}
      busy={busy}
      onCancel={clear}
      onConfirm={add}
    />
  )
}
