import { msg, plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import { type ComponentProps, memo, useState } from 'react'
import type { GameModsDiff } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  DismissGameModsFolder,
  UndismissGameModsFolders,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { i18n } from '../i18n/index.ts'
import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'
import { GameModsReviewDialog } from './GameModsReviewDialog.tsx'
import { ListCallout } from './ListCallout.tsx'
import { useGameModsDiff } from './useGameModsDiff.ts'

// Mod folders someone put straight into the game's Mods folder that the open profile lacks or holds at another
// version, offered for review. Folders Mortar cannot read are listed in the review but never raise the callout.
// The review reads the whole open profile, so only while it is open.
function ReviewNewFolders(props: Omit<ComponentProps<typeof GameModsReviewDialog>, 'profile'>) {
  const profile = useProfiles(openProfileOf)
  return profile ? <GameModsReviewDialog {...props} profile={profile} /> : null
}

export const NewFoldersCallout = memo(function NewFoldersCalloutView({
  profileId,
  updated,
}: {
  profileId: string
  updated: string
}) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game?.id ?? '')
  const [review, setReview] = useState<GameModsDiff | null>(null)
  const [pending, run] = usePending()
  const { diff, reload } = useGameModsDiff(game, profileId, updated)
  const missing = diff?.missing ?? []
  const different = diff?.different ?? []
  const count = missing.length + different.length
  const dismissAll = () =>
    run(
      async () => {
        const folders = [
          ...new Set([...missing.map((m) => m.folder ?? ''), ...different.map((m) => m.folder)]),
        ]
        await Promise.all(folders.map((f) => DismissGameModsFolder(game, f)))
        reload()
        useToasts.getState().push({
          kind: 'success',
          title: plural(folders.length, {
            one: 'Stopped asking about # folder',
            other: 'Stopped asking about # folders',
          }),
          action: {
            label: i18n._(msg`Undo`),
            run: () => UndismissGameModsFolders(game, folders).then(reload).catch(reportUnexpected),
          },
        })
      },
      { errorTitle: t`Could not dismiss those folders` },
    )
  return (
    <>
      {count === 0 ? null : (
        <ListCallout
          text={
            different.length > 0
              ? plural(count, {
                  one: "# mod in the game's Mods folder differs from this profile",
                  other: "# mods in the game's Mods folder differ from this profile",
                })
              : plural(count, {
                  one: "# mod in the game's Mods folder isn't in this profile",
                  other: "# mods in the game's Mods folder aren't in this profile",
                })
          }
          actions={
            <>
              <Button size="small" variant="text" disabled={pending} onClick={dismissAll}>
                {t`Don't ask about these`}
              </Button>
              <Button size="small" variant="contained" onClick={() => setReview(diff)}>
                {t`Review…`}
              </Button>
            </>
          }
        />
      )}
      {review === null ? null : (
        <ReviewNewFolders
          game={game}
          diff={review}
          onClose={() => {
            setReview(null)
            reload()
          }}
        />
      )}
    </>
  )
})
