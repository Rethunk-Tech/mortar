import { msg, plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import { useCallback, useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  DismissGameModsFolder,
  NewGameModsFolders,
  UndismissGameModsFolders,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { i18n } from '../i18n/index.ts'
import { useProfiles } from '../profiles/store.ts'
import { useFolderEvent } from '../shell/useFolderEvent.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'
import { ListCallout } from './ListCallout.tsx'
import { LockedReason } from './LockedReason.tsx'
import { MoveFoldersDialog } from './MoveFoldersDialog.tsx'
import { useLocked } from './useLocked.ts'
import { usePreviewRows } from './usePreviewRows.ts'

// Mod folders someone put straight into the game's Mods folder, offered for moving into this profile.
export function NewFoldersCallout({ profile }: { profile: Profile }) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game?.id ?? '')
  const locked = useLocked()
  const [open, setOpen] = useState(false)
  const [pending, run] = usePending()
  // A new profile.updated (an update, a move, a rollback) reads the folder again.
  const fetchRows = useCallback(
    () =>
      game === '' || profile.updated === ''
        ? Promise.resolve({ mods: [] })
        : NewGameModsFolders(game),
    [game, profile.updated],
  )
  const { mods, reload, drop } = usePreviewRows(true, fetchRows)
  useFolderEvent('library:mods-folder', game, reload)
  if (mods.length === 0) {
    return null
  }
  const dismissAll = () =>
    run(
      async () => {
        const folders = mods.map((m) => m.folder ?? '')
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
      <ListCallout
        text={plural(mods.length, {
          one: "# mod in the game's Mods folder isn't in Mortar.",
          other: "# mods in the game's Mods folder aren't in Mortar.",
        })}
        actions={
          <>
            <Button variant="text" disabled={pending} onClick={dismissAll}>
              {t`Don't ask about these`}
            </Button>
            <LockedReason locked={locked}>
              <Button variant="contained" disabled={locked} onClick={() => setOpen(true)}>
                {t`Move into this profile…`}
              </Button>
            </LockedReason>
          </>
        }
      />
      <MoveFoldersDialog
        open={open}
        game={game}
        profile={profile}
        mods={mods}
        onDismiss={drop}
        onClose={(moved) => {
          setOpen(false)
          if (moved) {
            reload()
          }
        }}
      />
    </>
  )
}
