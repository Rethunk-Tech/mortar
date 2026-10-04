import { useLingui } from '@lingui/react/macro'
import { FileJson } from 'lucide-react'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  ExportCollection,
  ShowExportedCollection,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/sharesvc/service.ts'
import { useProfiles } from '../profiles/store.ts'
import { MenuAction } from '../shell/MenuAction.tsx'
import { reportError, reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

export function ExportCollectionMenuItem({
  profile,
  close,
}: {
  profile: Profile
  close: () => void
}) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game)
  const exportDraft = async (gameId: string) => {
    const r = await ExportCollection(gameId, profile.id)
    if (!r.path) {
      return
    }
    const skipped = r.skipped ?? []
    const next = t`Nexus takes a 7z: run 7z x collection.zip -oc && 7z a collection.7z ./c/*, then upload it.`
    useToasts.getState().push({
      kind: skipped.length > 0 ? 'warning' : 'success',
      title: t`Collection draft saved`,
      body: skipped.length > 0 ? `${next}\n${t`Left out: ${skipped.join(', ')}`}` : next,
      action: {
        label: t`Show file`,
        run: () => ShowExportedCollection().catch(reportUnexpected),
      },
    })
  }
  return (
    <MenuAction
      icon={<FileJson size={16} />}
      label={t`Export as Nexus collection draft…`}
      disabled={!game}
      onClick={() => {
        close()
        if (game) {
          exportDraft(game.id).catch(reportError(t`Could not export the collection draft`))
        }
      }}
    />
  )
}
