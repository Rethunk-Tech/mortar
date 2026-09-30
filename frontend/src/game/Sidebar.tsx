import { Trans, useLingui } from '@lingui/react/macro'
import { alpha, Box, ButtonBase, IconButton } from '@mui/material'
import { ListOrdered, Plus, Settings } from 'lucide-react'
import { type PointerEvent, useState } from 'react'
import { openSettings, useNav } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { compact } from './compact.ts'
import { NewProfileDialog } from './NewProfileDialog.tsx'

const MIN = 150
const MAX = 300
const KEY = 'mortar.sidebarWidth'
const RAIL = 56

const clamp = (w: number) => Math.min(MAX, Math.max(MIN, w))

function storedWidth(): number {
  try {
    const n = Number(localStorage.getItem(KEY))
    return n ? clamp(n) : 220
  } catch {
    return 220
  }
}

function saveWidth(w: number) {
  try {
    localStorage.setItem(KEY, String(w))
  } catch {
    // Storage can be blocked; the width then lasts for this session only.
  }
}

export function Sidebar() {
  const { t } = useLingui()
  const allProfiles = useProfiles((s) => s.profiles)
  const profiles = allProfiles.filter((p) => !p.hidden)
  const openProfiles = useNav((s) => s.openProfiles)
  const openId = useProfiles((s) => s.openId)
  const open = useProfiles((s) => s.open)
  const [width, setWidth] = useState(storedWidth)
  const [creating, setCreating] = useState(false)
  const [drag, setDrag] = useState<{ x: number; w: number } | null>(null)

  const move = (e: PointerEvent<HTMLElement>) => {
    if (drag) {
      setWidth(clamp(drag.w + e.clientX - drag.x))
    }
  }
  const nudge = (delta: number) => {
    const w = clamp(width + delta)
    setWidth(w)
    saveWidth(w)
  }

  return (
    <Box
      component="nav"
      sx={{
        position: 'relative',
        width,
        flexShrink: 0,
        display: 'flex',
        flexDirection: 'column',
        borderRight: '2px solid',
        borderColor: 'primary.main',
        bgcolor: 'background.paper',
        [compact]: { width: RAIL },
      }}
    >
      <ButtonBase
        onClick={openProfiles}
        sx={{
          justifyContent: 'flex-start',
          px: 2,
          py: 1.25,
          fontSize: 14,
          fontWeight: 700,
          textTransform: 'uppercase',
          letterSpacing: 1,
          color: 'text.secondary',
          fontFamily: 'inherit',
          whiteSpace: 'nowrap',
          '&:hover': { bgcolor: 'action.hover' },
          [compact]: { display: 'none' },
        }}
      >
        <Trans>Profiles</Trans>
      </ButtonBase>
      <Box
        sx={{
          flex: 1,
          minHeight: 0,
          overflowY: 'auto',
          [compact]: { pt: 1, display: 'flex', flexDirection: 'column', alignItems: 'center' },
        }}
      >
        {profiles.map((p) => {
          const selected = p.id === openId
          return (
            <ButtonBase
              key={p.id}
              onClick={() => open(p.id)}
              aria-current={selected ? 'true' : undefined}
              aria-label={p.name}
              title={p.name}
              sx={{
                width: '100%',
                justifyContent: 'flex-start',
                px: 2,
                py: 1,
                fontFamily: 'inherit',
                fontSize: 15,
                textAlign: 'left',
                borderLeft: '3px solid',
                borderColor: selected ? 'primary.main' : 'transparent',
                bgcolor: (th) => (selected ? alpha(th.palette.primary.main, 0.16) : 'transparent'),
                '&:hover': { bgcolor: 'action.hover' },
                [compact]: {
                  width: 40,
                  height: 40,
                  mb: '4px',
                  p: 0,
                  justifyContent: 'center',
                  borderRadius: '6px',
                  border: '1px solid',
                  borderColor: selected ? 'primary.main' : 'transparent',
                },
              }}
            >
              <Box
                component="span"
                sx={{
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                  [compact]: { display: 'none' },
                }}
              >
                {p.name}
              </Box>
              <Box
                component="span"
                aria-hidden={true}
                sx={{ display: 'none', fontWeight: 700, [compact]: { display: 'inline' } }}
              >
                {[...p.name][0]?.toUpperCase()}
              </Box>
            </ButtonBase>
          )
        })}
        <ButtonBase
          onClick={() => setCreating(true)}
          sx={{
            width: '100%',
            justifyContent: 'flex-start',
            gap: 1,
            px: 2,
            py: 1,
            fontFamily: 'inherit',
            fontSize: 15,
            whiteSpace: 'nowrap',
            color: 'primary.main',
            '&:hover': { bgcolor: 'action.hover' },
            [compact]: { display: 'none' },
          }}
        >
          <Plus size={16} />
          <Trans>New profile</Trans>
        </ButtonBase>
      </Box>
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          p: 1,
          [compact]: { flexDirection: 'column', gap: 0.5 },
        }}
      >
        <IconButton
          aria-label={t`Manage profiles`}
          onClick={openProfiles}
          sx={{ display: 'none', [compact]: { display: 'inline-flex' } }}
        >
          <ListOrdered size={20} />
        </IconButton>
        <IconButton
          aria-label={t`New profile`}
          onClick={() => setCreating(true)}
          sx={{ display: 'none', [compact]: { display: 'inline-flex' } }}
        >
          <Plus size={20} />
        </IconButton>
        <IconButton aria-label={t`Settings`} onClick={() => openSettings()}>
          <Settings size={20} />
        </IconButton>
      </Box>
      <Box
        role="separator"
        aria-orientation="vertical"
        aria-label={t`Resize sidebar`}
        aria-valuemin={MIN}
        aria-valuemax={MAX}
        aria-valuenow={width}
        tabIndex={0}
        onPointerDown={(e) => {
          e.currentTarget.setPointerCapture(e.pointerId)
          setDrag({ x: e.clientX, w: width })
        }}
        onPointerMove={move}
        onPointerUp={() => {
          setDrag(null)
          saveWidth(width)
        }}
        onKeyDown={(e) => {
          if (e.key === 'ArrowLeft' || e.key === 'ArrowRight') {
            nudge(e.key === 'ArrowLeft' ? -16 : 16)
          }
        }}
        sx={{
          position: 'absolute',
          top: 0,
          bottom: 0,
          right: -6,
          width: 8,
          cursor: 'col-resize',
          zIndex: 1,
          [compact]: { display: 'none' },
        }}
      />
      <NewProfileDialog open={creating} onClose={() => setCreating(false)} />
    </Box>
  )
}
