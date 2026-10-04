import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import { useCallback, useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { NewGameModsFolders } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { useProfiles } from '../profiles/store.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { ListCallout } from './ListCallout.tsx'
import { MoveFoldersDialog } from './MoveFoldersDialog.tsx'
import { useLocked } from './useLocked.ts'
import { usePreviewRows } from './usePreviewRows.ts'

// Mod folders someone put straight into the game's Mods folder, offered for moving into this profile.
export function NewFoldersCallout({ profile }: { profile: Profile }) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game?.id ?? '')
  const locked = useLocked()
  const [open, setOpen] = useState(false)
  // A new profile.updated (an update, a move, a rollback) reads the folder again.
  const fetchRows = useCallback(
    () =>
      game === '' || profile.updated === ''
        ? Promise.resolve({ mods: [] })
        : NewGameModsFolders(game),
    [game, profile.updated],
  )
  const { mods, reload, drop } = usePreviewRows(true, fetchRows)
  if (mods.length === 0) {
    return null
  }
  return (
    <>
      <ListCallout
        text={plural(mods.length, {
          one: "# mod in the game's Mods folder isn't in Mortar.",
          other: "# mods in the game's Mods folder aren't in Mortar.",
        })}
        actions={
          <DisabledReason title={t`Stop the game to change mods.`} disabled={locked}>
            <Button variant="contained" disabled={locked} onClick={() => setOpen(true)}>
              {t`Move into this profile…`}
            </Button>
          </DisabledReason>
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
