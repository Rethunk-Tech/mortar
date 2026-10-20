import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import type { ReactNode } from 'react'
import type { RunError } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import { DismissRunError } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/service.ts'
import { useConsole } from '../console/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { download } from '../queue/actions.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { installableUpdate, sameId, updateFor } from './lookup.ts'
import { ReportToAuthorButton } from './ReportToAuthorButton.tsx'
import { useMods } from './store.ts'
import { openTarget } from './storeView.ts'
import { withOptionalLoaded } from './updateReview/wants.ts'
import { useUpdates } from './updates.ts'

export function RunErrorButtons({
  runError,
  button,
}: {
  runError: RunError
  button: (label: string, onClick: () => void) => ReactNode
}) {
  const { t } = useLingui()
  const mods = useMods((s) => s.mods)
  const setEnabled = useMods((s) => s.setEnabled)
  const profileId = useProfiles((s) => s.openId)
  const mod = mods.find((m) => m.key === runError.key && sameId(m.id, runError.id))
  const current =
    mod ?? (runError.updated ? mods.find((m) => sameId(m.id, runError.id)) : undefined)
  const profile = useProfiles((s) => s.profiles.find((p) => p.id === profileId))
  const updates = useUpdates((s) => s.updates)
  const update = runError.updated && current ? updateFor(updates, current, profile) : undefined
  const installable = update && installableUpdate(update) ? update : undefined
  const openHelp = () => {
    const at = openTarget()
    if (!at || runError.runId === '') {
      return
    }
    useConsole.getState().viewRun(at.game, at.id, runError.runId)
    useConsole.getState().setHelping(true)
  }
  const gameId = useProfiles((s) => s.game?.id ?? '')
  return (
    <>
      {current && !runError.updated
        ? button(t`Disable`, () => setEnabled(current, false).catch(reportUnexpected))
        : null}
      {installable
        ? button(t`Update`, () =>
            withOptionalLoaded(installable, profile).then(download).catch(reportUnexpected),
          )
        : null}
      {current && profile && gameId && runError.runId !== '' ? (
        <ReportToAuthorButton
          game={gameId}
          profile={profile}
          runId={runError.runId}
          mod={current}
          modLogName={runError.name}
        />
      ) : null}
      <Button
        size="small"
        color="warning"
        variant="outlined"
        onClick={openHelp}
        sx={{ flexShrink: 0 }}
      >
        {t`Show in log`}
      </Button>
      <Button
        size="small"
        color="inherit"
        variant="text"
        onClick={() => {
          const at = openTarget()
          if (at) {
            DismissRunError(at.game, at.id, runError.key)
              .then(() => useMods.getState().loadProblems())
              .catch(reportUnexpected)
          }
        }}
        sx={{ flexShrink: 0 }}
      >
        {t`Dismiss`}
      </Button>
    </>
  )
}
