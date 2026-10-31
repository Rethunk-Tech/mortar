import { useLingui } from '@lingui/react/macro'
import { Trust } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import { installExtraFolder } from '../downloads/autoInstall.ts'
import { persist } from '../settings/persist.ts'
import { ConfirmationDialog } from '../shell/dialog/ConfirmationDialog.tsx'
import { buildProject, packProject, useTrustAsk } from './builds.ts'
import { useDevKinds } from './kinds.ts'

/** Asks once, before a project from outside the New mod wizard first builds, because a build runs its code. */
export function TrustDialog() {
  const { t } = useLingui()
  const ask = useTrustAsk((s) => s.ask)
  const close = () => useTrustAsk.setState({ ask: null })
  const isAsset = useDevKinds((s) => Boolean(ask && s.byDir[ask.dir]))
  return (
    <ConfirmationDialog
      open={ask !== null}
      tone="warning"
      title={isAsset ? t`Install this project?` : t`Build this project?`}
      body={
        isAsset
          ? t`Its files install into the profile as they are, and a plugin among them runs when the game starts. Install it?`
          : t`Building runs this repository's code on your computer. Build it?`
      }
      confirmLabel={isAsset ? t`Install` : t`Build`}
      onCancel={close}
      onConfirm={() => {
        if (!ask) {
          return
        }
        close()
        persist(async () => {
          await Trust(ask.game, ask.dir)
          if (isAsset) {
            // The folder watch does not fire for a folder that already exists, so the swap to the project folder installs here.
            installExtraFolder(ask.game, ask.dir)
          }
          await (ask.pack ? packProject(ask.game, ask.dir) : buildProject(ask.game, ask.dir))
        }, t`Could not trust the project`)
      }}
    />
  )
}
