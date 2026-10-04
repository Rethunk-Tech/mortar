import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
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
      <Typography sx={heading}>{t`Hidden inside this mod`}</Typography>
      {mine.map((h) => (
        <Box key={`${h.folder}-${h.uniqueId}`} sx={{ py: 0.25 }}>
          <Typography sx={{ fontSize: 13, overflowWrap: 'anywhere' }}>
            {h.version === '' ? h.name : `${h.name} · ${h.version}`}
          </Typography>
          <Typography sx={{ fontSize: 12, color: 'text.secondary', overflowWrap: 'anywhere' }}>
            {h.folder}
          </Typography>
        </Box>
      ))}
      <Typography sx={{ fontSize: 12, color: 'text.secondary', mt: 0.5 }}>
        {t`SMAPI skips folders whose names start with a dot.`}
      </Typography>
    </Box>
  )
}
