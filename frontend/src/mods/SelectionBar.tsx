import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { Power, PowerOff, Share2, Trash2, X } from 'lucide-react'
import type { Mod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { openShare } from '../share/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { modId } from './lookup.ts'
import { useSelection } from './selection.ts'
import { useMods } from './store.ts'
import { useLocked } from './useLocked.ts'

const noWrap = { whiteSpace: 'nowrap' } as const

export function SelectionBar({ profileId, mods }: { profileId: string; mods: Mod[] }) {
  const { t } = useLingui()
  const ids = useSelection((s) => s.ids)
  const clear = useSelection((s) => s.clear)
  const setEnabledMany = useMods((s) => s.setEnabledMany)
  const askRemove = useMods((s) => s.askRemove)
  const locked = useLocked()
  if (ids.length < 2) {
    return null
  }
  const selected = mods.filter((m) => ids.includes(modId(m)))
  const keys = [...new Set(selected.map((m) => m.key))]
  const count = plural(ids.length, { one: '# mod selected', other: '# mods selected' })
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 1,
        px: 2,
        py: 0.75,
        minHeight: 40,
        borderBottom: '1px solid rgba(255,255,255,0.08)',
        flexWrap: 'wrap',
      }}
    >
      <Typography sx={{ fontSize: 13, mr: 0.5, whiteSpace: 'nowrap' }}>{count}</Typography>
      <Button
        size="small"
        variant="outlined"
        disabled={locked}
        startIcon={<Power size={15} />}
        onClick={() => setEnabledMany(selected, true).catch(reportUnexpected)}
        sx={noWrap}
      >
        {t`Enable`}
      </Button>
      <Button
        size="small"
        variant="outlined"
        disabled={locked}
        startIcon={<PowerOff size={15} />}
        onClick={() => setEnabledMany(selected, false).catch(reportUnexpected)}
        sx={noWrap}
      >
        {t`Disable`}
      </Button>
      <Button
        size="small"
        variant="outlined"
        color="error"
        disabled={locked}
        startIcon={<Trash2 size={15} />}
        onClick={() => askRemove(selected)}
        sx={noWrap}
      >
        {t`Remove`}
      </Button>
      <Button
        size="small"
        variant="outlined"
        startIcon={<Share2 size={15} />}
        onClick={() => openShare(profileId, keys)}
        sx={noWrap}
      >
        {t`Share selection`}
      </Button>
      <Button size="small" variant="text" startIcon={<X size={15} />} onClick={clear} sx={noWrap}>
        {t`Clear`}
      </Button>
    </Box>
  )
}
