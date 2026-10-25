import { useLingui } from '@lingui/react/macro'
import { Link } from '@mui/material'
import { useState } from 'react'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { CopyModsWithNeeds } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { useProfiles } from '../profiles/store.ts'
import { OtherProfilesDialog } from './OtherProfilesDialog.tsx'
import { rowLink } from './paper.ts'

// The Also add to… link and its dialog: opening it changes only this, not the panel around it.
export function AlsoAdd({ mod, profile }: { mod: Mod; profile: Profile }) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game?.id ?? '')
  const [open, setOpen] = useState(false)
  return (
    <>
      <Link component="button" onClick={() => setOpen(true)} sx={rowLink}>
        {t`Also add to…`}
      </Link>
      <OtherProfilesDialog
        open={open}
        onClose={() => setOpen(false)}
        game={game}
        currentProfileId={profile.id}
        id={mod.id}
        title={t`Also add ${{ name: mod.name }} to…`}
        withNeeds={true}
        confirmLabel={t`Add`}
        onConfirm={async (profiles) => {
          await Promise.all(
            profiles.map((other) => CopyModsWithNeeds(game, profile.id, other.id, [mod.id])),
          )
        }}
      />
    </>
  )
}
