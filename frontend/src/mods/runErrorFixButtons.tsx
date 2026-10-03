import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import type { ReactNode } from 'react'
import type { RunError } from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import { useConsole } from '../console/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { download } from '../queue/actions.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { installableUpdate, sameId, updateFor } from './lookup.ts'
import { ReportToAuthorButton } from './ReportToAuthorButton.tsx'
import { useMods } from './store.ts'
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
  const mod = mods.find((m) => m.key === runError.key && sameId(m.uniqueId, runError.uniqueId))
  const current =
    mod ?? (runError.updated ? mods.find((m) => sameId(m.uniqueId, runError.uniqueId)) : undefined)
  const profile = useProfiles.getState().profiles.find((p) => p.id === profileId)
  const update =
    runError.updated && current
      ? updateFor(useUpdates.getState().updates, current, profile)
      : undefined
  const want =
    update && installableUpdate(update)
      ? {
          kind: 'update' as const,
          ...(update.githubRepo ? { repo: update.githubRepo } : { modId: update.nexusId }),
          name: update.name,
          version: update.version,
          currentKey: update.key,
        }
      : undefined
  const openHelp = () => {
    const { game, openId } = useProfiles.getState()
    if (!(game && openId) || runError.runId === '') {
      return
    }
    useConsole.getState().viewRun(game.id, openId, runError.runId)
    useConsole.getState().setHelping(true)
  }
  const gameId = useProfiles((s) => s.game?.id ?? '')
  return (
    <>
      {current && !runError.updated
        ? button(t`Switch off`, () => setEnabled(current, false).catch(reportUnexpected))
        : null}
      {want ? button(t`Update`, () => download([want]).catch(reportUnexpected)) : null}
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
        sx={{ height: 28, whiteSpace: 'nowrap', flexShrink: 0 }}
      >
        {t`Show in log`}
      </Button>
    </>
  )
}
