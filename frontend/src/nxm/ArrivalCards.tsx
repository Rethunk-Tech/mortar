import { useLingui } from '@lingui/react/macro'
import { Box, Button, Menu, MenuItem } from '@mui/material'
import { useEffect, useState } from 'react'
import type { Arrival } from '../../bindings/github.com/Rethunk-AI/mortar/internal/nxmsvc/models.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { List } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { Logo } from '../brand/Logo.tsx'
import { useSettings } from '../settings/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { NXM_GAME, useNxm } from './store.ts'

function ArrivalCard({ arrival }: { arrival: Arrival }) {
  const { t } = useLingui()
  const choose = useNxm((s) => s.choose)
  const dismiss = useNxm((s) => s.dismiss)
  const lastId = useSettings((s) => s.lastProfile?.[NXM_GAME])
  const [profiles, setProfiles] = useState<Profile[]>([])
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  useEffect(() => {
    List(NXM_GAME)
      .then((list) => setProfiles((list ?? []).filter((p) => !p.hidden)))
      .catch(reportUnexpected)
  }, [])
  const open = profiles.find((p) => p.id === lastId) ?? profiles[0]
  const others = profiles.filter((p) => p.id !== open?.id)
  return (
    <Box
      role="alertdialog"
      aria-label={t`Unexpected Nexus link`}
      sx={{
        display: 'flex',
        gap: '12px',
        p: '14px',
        bgcolor: 'rgba(36,36,40,0.97)',
        border: '1px solid rgba(255,255,255,0.12)',
        borderRadius: '12px',
      }}
    >
      <Box
        sx={{
          width: 40,
          height: 40,
          flexShrink: 0,
          display: 'grid',
          placeItems: 'center',
          borderRadius: '10px',
          bgcolor: 'primary.main',
        }}
      >
        <Logo size={22} />
      </Box>
      <Box sx={{ flexGrow: 1, minWidth: 0, display: 'flex', flexDirection: 'column', gap: '8px' }}>
        <Box component="span" sx={{ fontSize: 14, fontWeight: 700 }}>
          {t`Mortar · Install Nexus mod ${arrival.link.modId}?`}
        </Box>
        <Box component="span" sx={{ fontSize: 14 }}>
          {t`You started this download on Nexus. Choose the profile it goes into.`}
        </Box>
        <Box sx={{ display: 'flex', gap: 1 }}>
          {open ? (
            <Button
              variant="contained"
              size="small"
              onClick={() => choose(arrival.id, open.id)}
              sx={{ whiteSpace: 'nowrap' }}
            >
              {open.name}
            </Button>
          ) : null}
          {others.length > 0 ? (
            <Button
              variant="outlined"
              size="small"
              onClick={(e) => setAnchor(e.currentTarget)}
              sx={{ whiteSpace: 'nowrap' }}
            >
              {t`Other profile…`}
            </Button>
          ) : null}
          <Button size="small" onClick={() => dismiss(arrival.id)} sx={{ whiteSpace: 'nowrap' }}>
            {t`Ignore`}
          </Button>
        </Box>
        <Menu anchorEl={anchor} open={anchor !== null} onClose={() => setAnchor(null)}>
          {others.map((p) => (
            <MenuItem
              key={p.id}
              onClick={() => {
                setAnchor(null)
                choose(arrival.id, p.id)
              }}
            >
              {p.name}
            </MenuItem>
          ))}
        </Menu>
      </Box>
    </Box>
  )
}

export function ArrivalCards() {
  const arrivals = useNxm((s) => s.arrivals)
  return (
    <Box
      sx={{
        position: 'fixed',
        top: 44,
        left: '50%',
        transform: 'translateX(-50%)',
        width: 460,
        display: 'flex',
        flexDirection: 'column',
        gap: '10px',
        zIndex: 'tooltip',
      }}
    >
      {arrivals.map((a) => (
        <ArrivalCard key={a.id} arrival={a} />
      ))}
    </Box>
  )
}
