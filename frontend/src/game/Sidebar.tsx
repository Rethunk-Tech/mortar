import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, IconButton } from '@mui/material'
import { ChevronRight, ListOrdered, Plus } from 'lucide-react'
import { type PointerEvent, useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { PlayControl } from '../launch/PlayControl.tsx'
import { useNav } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { compact } from './compact.ts'
import { NewProfileDialog } from './NewProfileDialog.tsx'

const MIN = 150
const MAX = 300
const KEY = 'mortar.sidebarWidth'
const RAIL = 56
const DEFAULT_WIDTH = 220
const NUDGE_PX = 16
const SELECTED_WEIGHT = 600
const INITIALS = 2
const WHITESPACE = /\s+/

const clamp = (w: number) => Math.min(MAX, Math.max(MIN, w))

function storedWidth(): number {
  try {
    const n = Number(localStorage.getItem(KEY))
    return n ? clamp(n) : DEFAULT_WIDTH
  } catch {
    return DEFAULT_WIDTH
  }
}

function saveWidth(w: number) {
  try {
    localStorage.setItem(KEY, String(w))
  } catch {
    // Storage can be blocked; the width then lasts for this session only.
  }
}

function initials(name: string): string {
  const words = name.trim().split(WHITESPACE)
  const chars =
    words.length > 1
      ? words.slice(0, INITIALS).map((w) => [...w][0] ?? '')
      : [...name].slice(0, INITIALS)
  return chars.join('').toUpperCase()
}

function ProfileButton({
  profile,
  selected,
  onOpen,
}: {
  profile: Profile
  selected: boolean
  onOpen: () => void
}) {
  return (
    <ButtonBase
      onClick={onOpen}
      aria-current={selected ? 'true' : undefined}
      aria-label={profile.name}
      title={profile.name}
      sx={{
        width: '100%',
        height: 40,
        justifyContent: 'flex-start',
        px: '10px',
        borderRadius: '6px',
        fontFamily: 'inherit',
        fontSize: 14,
        fontWeight: selected ? SELECTED_WEIGHT : 'normal',
        textAlign: 'left',
        color: selected ? '#ffffff' : 'rgba(255,255,255,0.88)',
        bgcolor: selected ? 'rgba(255,255,255,0.12)' : 'transparent',
        '&:hover': { bgcolor: selected ? 'rgba(255,255,255,0.12)' : 'action.hover' },
        [compact]: {
          width: 40,
          p: 0,
          justifyContent: 'center',
          borderRadius: '8px',
          bgcolor: selected ? 'rgba(255,255,255,0.14)' : 'transparent',
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
        {profile.name}
      </Box>
      <Box
        component="span"
        aria-hidden={true}
        sx={{
          display: 'none',
          fontWeight: 700,
          color: '#ffffff',
          [compact]: { display: 'inline' },
        }}
      >
        {initials(profile.name)}
      </Box>
    </ButtonBase>
  )
}

function ResizeHandle({ width, onWidth }: { width: number; onWidth: (w: number) => void }) {
  const { t } = useLingui()
  const [drag, setDrag] = useState<{ x: number; w: number } | null>(null)
  const move = (e: PointerEvent<HTMLElement>) => {
    if (drag) {
      onWidth(clamp(drag.w + e.clientX - drag.x))
    }
  }
  const nudge = (delta: number) => {
    const w = clamp(width + delta)
    onWidth(w)
    saveWidth(w)
  }
  return (
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
          nudge(e.key === 'ArrowLeft' ? -NUDGE_PX : NUDGE_PX)
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
  )
}

export function Sidebar({ game }: { game: string }) {
  const { t } = useLingui()
  const allProfiles = useProfiles((s) => s.profiles)
  const profiles = allProfiles.filter((p) => !p.hidden)
  const openProfiles = useNav((s) => s.openProfiles)
  const openId = useProfiles((s) => s.openId)
  const open = useProfiles((s) => s.open)
  const [width, setWidth] = useState(storedWidth)
  const [creating, setCreating] = useState(false)

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
          justifyContent: 'space-between',
          height: 40,
          flexShrink: 0,
          px: '14px',
          fontSize: 12,
          fontWeight: 700,
          textTransform: 'uppercase',
          letterSpacing: '0.08em',
          color: 'text.secondary',
          fontFamily: 'inherit',
          whiteSpace: 'nowrap',
          '&:hover': { bgcolor: 'action.hover' },
          [compact]: { display: 'none' },
        }}
      >
        {t`Profiles`}
        <ChevronRight size={14} aria-hidden={true} />
      </ButtonBase>
      <Box
        sx={{
          flex: 1,
          minHeight: 0,
          overflowY: 'auto',
          px: '6px',
          display: 'flex',
          flexDirection: 'column',
          gap: '2px',
          [compact]: {
            pt: 1,
            px: 0,
            gap: 0,
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
          },
        }}
      >
        {profiles.map((p) => (
          <ProfileButton
            key={p.id}
            profile={p}
            selected={p.id === openId}
            onOpen={() => open(p.id)}
          />
        ))}
        <ButtonBase
          onClick={() => setCreating(true)}
          sx={{
            width: '100%',
            height: 40,
            flexShrink: 0,
            justifyContent: 'flex-start',
            gap: 1,
            px: '10px',
            borderRadius: '6px',
            fontFamily: 'inherit',
            fontSize: 14,
            whiteSpace: 'nowrap',
            color: 'primary.main',
            '&:hover': { bgcolor: 'action.hover' },
            [compact]: { display: 'none' },
          }}
        >
          <Plus size={16} />
          {t`New profile`}
        </ButtonBase>
      </Box>
      <Box
        sx={{
          display: 'none',
          [compact]: {
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            gap: 0.5,
            pt: 1,
          },
        }}
      >
        <IconButton aria-label={t`Manage profiles`} onClick={openProfiles}>
          <ListOrdered size={20} />
        </IconButton>
        <IconButton aria-label={t`New profile`} onClick={() => setCreating(true)}>
          <Plus size={20} />
        </IconButton>
      </Box>
      <Box
        sx={{
          borderTop: '1px solid rgba(255,255,255,0.08)',
          [compact]: { display: 'flex', borderTop: 0 },
        }}
      >
        <PlayControl game={game} />
      </Box>
      <ResizeHandle width={width} onWidth={setWidth} />
      <NewProfileDialog open={creating} onClose={() => setCreating(false)} />
    </Box>
  )
}
