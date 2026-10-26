import { useLingui } from '@lingui/react/macro'
import { Trust } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import { persist } from '../settings/persist.ts'
import { ConfirmationDialog } from '../shell/dialog/ConfirmationDialog.tsx'
import { buildProject, packProject, useTrustAsk } from './builds.ts'

/** Asks once, before a project from outside the New mod wizard first builds, because a build runs its code. */
export function TrustDialog() {
  const { t } = useLingui()
  const ask = useTrustAsk((s) => s.ask)
  const close = () => useTrustAsk.setState({ ask: null })
  return (
    <ConfirmationDialog
      open={ask !== null}
      tone="warning"
      title={t`Build this project?`}
      body={t`Building runs this repository's code on your computer. Build it?`}
      confirmLabel={t`Build`}
      onCancel={close}
      onConfirm={() => {
        if (!ask) {
          return
        }
        close()
        persist(async () => {
          await Trust(ask.game, ask.dir)
          await (ask.pack ? packProject(ask.game, ask.dir) : buildProject(ask.game, ask.dir))
        }, t`Could not save that setting`)
      }}
    />
  )
}
