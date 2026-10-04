import { useLingui } from '@lingui/react/macro'
import { Box, List, ListItem, ListItemText, Typography } from '@mui/material'
import { useEffect, useState } from 'react'
import type {
  HiddenMod,
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { DotHiddenMods } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { heading } from './paper.ts'

// Mods SMAPI skips because a folder on their path starts with a dot, listed under the entry that holds them.
export function HiddenInside({ mod, profile }: { mod: Mod; profile: Profile }) {
  const { t } = useLingui()
  const on = useSettings((s) => s.games?.stardew?.showDotHiddenMods === true)
  const game = useProfiles((s) => s.game?.id ?? '')
  const [hidden, setHidden] = useState<HiddenMod[]>([])
  useEffect(() => {
    if (!on || game === '' || profile.updated === '') {
      setHidden([])
      return
    }
    let live = true
    DotHiddenMods(game, profile.id)
      .then((list) => live && setHidden(list ?? []))
      .catch(reportUnexpected)
    return () => {
      live = false
    }
  }, [on, game, profile.id, profile.updated])
  const mine = hidden.filter((h) => h.key === mod.key)
  if (mine.length === 0) {
    return null
  }
  return (
    <Box>
      <Typography component="h3" sx={heading}>
        {t`Hidden inside this mod`}
      </Typography>
      <List dense={true} disablePadding={true}>
        {mine.map((h) => (
          <ListItem key={`${h.folder}-${h.uniqueId}`} disableGutters={true} sx={{ py: 0.25 }}>
            <ListItemText
              primary={h.version === '' ? h.name : `${h.name} · ${h.version}`}
              secondary={h.folder}
              slotProps={{
                primary: { sx: { fontSize: 13, overflowWrap: 'anywhere' } },
                secondary: { sx: { fontSize: 12, overflowWrap: 'anywhere' } },
              }}
            />
          </ListItem>
        ))}
      </List>
      <Typography sx={{ fontSize: 12, color: 'text.secondary', mt: 0.5 }}>
        {t`SMAPI skips folders whose names start with a dot.`}
      </Typography>
    </Box>
  )
}
