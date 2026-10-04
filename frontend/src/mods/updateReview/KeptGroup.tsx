import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { ChevronDown, ChevronRight, Pin } from 'lucide-react'
import { useState } from 'react'
import type { Update } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import type { Mod } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import { sameId } from '../lookup.ts'
import { useMods } from '../store.ts'
import { Version } from './Version.tsx'

/** Pinned mods that have a newer version, collapsed, each with Unpin. */
export function KeptGroup({ kept, mods }: { kept: Update[]; mods: Mod[] }) {
  const { t } = useLingui()
  const setPinned = useMods((s) => s.setPinned)
  const [open, setOpen] = useState(false)
  if (kept.length === 0) {
    return null
  }
  return (
    <Box>
      <Button
        aria-expanded={open}
        startIcon={open ? <ChevronDown size={16} /> : <ChevronRight size={16} />}
        onClick={() => setOpen(!open)}
        sx={{ px: 3, py: 1.25, width: '100%', justifyContent: 'flex-start' }}
      >
        {t`Kept at this version (${kept.length})`}
      </Button>
      {open ? (
        <Box role="list">
          {kept.map((u) => {
            const mod = mods.find((m) => m.key === u.key && sameId(m.uniqueId, u.uniqueId))
            return (
              <Box
                key={u.key}
                role="listitem"
                sx={{ display: 'flex', alignItems: 'center', gap: 1.5, px: 3, py: 1 }}
              >
                <Pin size={14} aria-hidden={true} />
                <Typography noWrap={true} sx={{ flex: 1, minWidth: 0 }}>
                  {u.name}
                </Typography>
                <Version>{u.installed}</Version>
                <Typography sx={{ fontSize: 13, color: 'text.secondary' }}>
                  {t`newer: ${u.version}`}
                </Typography>
                {mod ? (
                  <Button onClick={() => setPinned(mod, false).catch(reportUnexpected)}>
                    {t`Unpin`}
                  </Button>
                ) : null}
              </Box>
            )
          })}
        </Box>
      ) : null}
    </Box>
  )
}
