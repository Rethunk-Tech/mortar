import { msg, plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import { useState } from 'react'
import type {
  GameModsDiff,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  DismissGameModsFolder,
  UndismissGameModsFolders,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { i18n } from '../i18n/index.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'
import { GameModsReviewDialog } from './GameModsReviewDialog.tsx'
import { ListCallout } from './ListCallout.tsx'
import { useGameModsDiff } from './useGameModsDiff.ts'

// Mod folders someone put straight into the game's Mods folder that the open profile lacks or holds at another
// version, offered for review. Folders Mortar cannot read are listed in the review but never raise the callout.
export function NewFoldersCallout({ profile }: { profile: Profile }) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game?.id ?? '')
  const [review, setReview] = useState<GameModsDiff | null>(null)
  const [pending, run] = usePending()
  const { diff, reload } = useGameModsDiff(game, profile.id, profile.updated)
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
              <Button variant="text" disabled={pending} onClick={dismissAll}>
                {t`Don't ask about these`}
              </Button>
              <Button variant="contained" onClick={() => setReview(diff)}>
                {t`Review…`}
              </Button>
            </>
          }
        />
      )}
      {review === null ? null : (
        <GameModsReviewDialog
          game={game}
          profile={profile}
          diff={review}
          onClose={() => {
            setReview(null)
            reload()
          }}
        />
      )}
    </>
  )
}
