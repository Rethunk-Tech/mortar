import { useLingui } from '@lingui/react/macro'
import { Trust } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import { saveSetting } from '../settings/persist.ts'
import { useSettings } from '../settings/store.ts'
import { ConfirmationDialog } from '../shell/dialog/ConfirmationDialog.tsx'
import { buildProject, packProject, useTrustAsk } from './builds.ts'

/** Asks once, before a project from outside the New mod wizard first builds, because a build runs its code. */
export function TrustDialog() {
  const { t } = useLingui()
  const ask = useTrustAsk((s) => s.ask)
  const close = () => useTrustAsk.setState({ ask: null })
  const projects = useSettings((s) => (ask ? s.games?.[ask.game]?.devProjects : undefined))
  const asset = (projects ?? []).some((p) => p.dir === ask?.dir && p.kind)
  return (
    <ConfirmationDialog
      open={ask !== null}
      tone="warning"
      title={asset ? t`Install this project?` : t`Build this project?`}
      body={
        asset
          ? t`Its files install into the profile as they are, and a plugin among them runs when the game starts. Install it?`
          : t`Building runs this repository's code on your computer. Build it?`
      }
      confirmLabel={asset ? t`Install` : t`Build`}
      onCancel={close}
      onConfirm={() => {
        if (!ask) {
          return
        }
        close()
        saveSetting(async () => {
          await Trust(ask.game, ask.dir)
          await (ask.pack ? packProject(ask.game, ask.dir) : buildProject(ask.game, ask.dir))
        })
      }}
    />
  )
}
