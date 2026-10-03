import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, IconButton, Tooltip } from '@mui/material'
import { ChevronRight, ListOrdered, Plus } from 'lucide-react'
import { type PointerEvent, useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { HelpDialog } from '../console/HelpDialog.tsx'
import { PlayControl } from '../launch/PlayControl.tsx'
import { useBadges } from '../mods/badges.ts'
import { ProfileHealth } from '../mods/ProfileHealth.tsx'
import { useNav } from '../nav/store.ts'
import { ProfileMark } from '../profiles/ProfileMark.tsx'
import { RecentChangesButton } from '../profiles/RecentChangesButton.tsx'
import { useProfiles } from '../profiles/store.ts'
import { QueueButton } from '../queue/QueueButton.tsx'
import { HistoryButton } from '../toasts/HistoryButton.tsx'
import { compact } from './compact.ts'
import { NewProfileDialog } from './NewProfileDialog.tsx'
import { ProfileContextMenu } from './ProfileContextMenu.tsx'
import { SupportButton } from './SupportButton.tsx'
import { useSidebarCollapsed } from './sidebarCollapsed.ts'
import { useTab } from './tab.ts'
import { useOrderedProfiles, useSidebarBadges } from './useSidebarProfiles.ts'

const MIN = 150
const MAX = 300
const KEY = 'mortar.sidebarWidth'
const RAIL = 56
const DEFAULT_WIDTH = 220
const NUDGE_PX = 16
const SELECTED_WEIGHT = 600
const INITIALS = 2
const WHITESPACE = /\s+/

const rail = (sx: object) => ({ [compact]: sx, '[data-collapsed="true"] &': sx })

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

function Badges({ profile }: { profile: Profile }) {
  const counts = useBadges((s) => s.byProfile[profile.id])
  return (
    <ProfileHealth
      counts={counts}
      sidebar={true}
      onClick={() => useTab.getState().setTab('problems')}
    />
  )
}

function ProfileButton({
  profile,
  selected,
  onOpen,
  onMenu,
}: {
  profile: Profile
  selected: boolean
  onOpen: () => void
  onMenu: (position: { top: number; left: number }) => void
}) {
  return (
    <ButtonBase
      onClick={onOpen}
      onContextMenu={(e) => {
        e.preventDefault()
        onMenu({ top: e.clientY, left: e.clientX })
      }}
      onKeyDown={(e) => {
        if (e.key === 'ContextMenu' || (e.key === 'F10' && e.shiftKey)) {
          e.preventDefault()
          const rect = e.currentTarget.getBoundingClientRect()
          onMenu({ top: rect.bottom, left: rect.left })
        }
      }}
      aria-current={selected ? 'true' : undefined}
      aria-label={profile.name}
      title={profile.name}
      sx={{
        width: '100%',
        height: 40,
        justifyContent: 'flex-start',
        px: '10px',
        position: 'relative',
        borderRadius: '6px',
        fontFamily: 'inherit',
        fontSize: 14,
        fontWeight: selected ? SELECTED_WEIGHT : 'normal',
        textAlign: 'left',
        color: selected ? '#ffffff' : 'rgba(255,255,255,0.88)',
        bgcolor: selected ? 'rgba(255,255,255,0.12)' : 'transparent',
        '&:hover': { bgcolor: selected ? 'rgba(255,255,255,0.12)' : 'action.hover' },
        ...rail({
          width: 40,
          p: 0,
          justifyContent: 'center',
          borderRadius: '8px',
          bgcolor: selected ? 'rgba(255,255,255,0.14)' : 'transparent',
        }),
      }}
    >
      <Box
        component="span"
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: 1,
          flex: 1,
          minWidth: 0,
          overflow: 'hidden',
          ...rail({ display: 'none' }),
        }}
      >
        <ProfileMark profile={profile} size={22} />
        <Box
          component="span"
          sx={{
            minWidth: 0,
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
          }}
        >
          {profile.name}
        </Box>
      </Box>
      <Badges profile={profile} />
      <Box
        component="span"
        aria-hidden={true}
        sx={{
          display: 'none',
          fontWeight: 700,
          color: '#ffffff',
          ...rail({ display: 'inline-flex' }),
        }}
      >
        {profile.color || profile.icon ? (
          <ProfileMark profile={profile} size={28} />
        ) : (
          initials(profile.name)
        )}
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
        ...rail({ display: 'none' }),
      }}
    />
  )
}

// Downloads, recent changes, notifications and Support over the Play button, pinned under the profile list.
function BottomBlock({ game }: { game: string }) {
  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        borderTop: '1px solid rgba(255,255,255,0.08)',
        ...rail({ borderTop: 0 }),
      }}
    >
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          height: 40,
          pr: '6px',
          '& > button, & .MuiIconButton-root': { height: 40, width: 40 },
          ...rail({ flexDirection: 'column', pr: 0 }),
        }}
      >
        <Box sx={{ flex: 1, height: 40, display: 'flex', alignItems: 'center' }}>
          <QueueButton />
        </Box>
        <RecentChangesButton game={game} />
        <HistoryButton />
        <SupportButton game={game} />
      </Box>
      <PlayControl game={game} />
    </Box>
  )
}

// ProfileList is the sidebar's profile buttons, each with the profile page's actions on right-click. The menu stays
// mounted after it closes so the dialogs it opened stay open.
function ProfileList({ game, profiles }: { game: string; profiles: Profile[] }) {
  const openId = useProfiles((s) => s.openId)
  const open = useProfiles((s) => s.open)
  const [menu, setMenu] = useState<{
    id: string
    position: { top: number; left: number } | null
  } | null>(null)
  const menuProfile = menu ? profiles.find((p) => p.id === menu.id) : undefined
  return (
    <>
      {profiles.map((p) => (
        <ProfileButton
          key={p.id}
          profile={p}
          selected={p.id === openId}
          onOpen={() => open(p.id)}
          onMenu={(position) => setMenu({ id: p.id, position })}
        />
      ))}
      {menuProfile ? (
        <ProfileContextMenu
          game={game}
          profile={menuProfile}
          position={menu?.position ?? null}
          onClose={() => setMenu((m) => (m ? { ...m, position: null } : m))}
        />
      ) : null}
    </>
  )
}

export function Sidebar({ game }: { game: string }) {
  const { t } = useLingui()
  const { profiles } = useOrderedProfiles(game)
  useSidebarBadges(game)
  const openProfiles = useNav((s) => s.openProfiles)
  const [width, setWidth] = useState(storedWidth)
  const [creating, setCreating] = useState(false)
  const collapsed = useSidebarCollapsed((s) => s.collapsed)

  return (
    <Box
      component="nav"
      data-collapsed={collapsed ? 'true' : undefined}
      sx={{
        position: 'relative',
        width: collapsed ? RAIL : width,
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
          ...rail({ display: 'none' }),
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
          ...rail({
            pt: 1,
            px: 0,
            gap: 0,
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
          }),
        }}
      >
        <ProfileList game={game} profiles={profiles} />
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
            ...rail({ display: 'none' }),
          }}
        >
          <Plus size={16} aria-hidden={true} />
          {t`New profile`}
        </ButtonBase>
      </Box>
      <Box
        sx={{
          display: 'none',
          ...rail({
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            gap: 0.5,
            pt: 1,
          }),
        }}
      >
        <Tooltip title={t`Manage profiles`}>
          <IconButton aria-label={t`Manage profiles`} onClick={openProfiles}>
            <ListOrdered size={20} />
          </IconButton>
        </Tooltip>
        <Tooltip title={t`New profile`}>
          <IconButton aria-label={t`New profile`} onClick={() => setCreating(true)}>
            <Plus size={20} />
          </IconButton>
        </Tooltip>
      </Box>
      <BottomBlock game={game} />
      <ResizeHandle width={width} onWidth={setWidth} />
      <NewProfileDialog open={creating} onClose={() => setCreating(false)} />
      <HelpDialog game={game} />
    </Box>
  )
}
