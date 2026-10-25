import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase } from '@mui/material'
import { ChevronDown, Download, LayoutTemplate, ListOrdered, Plus } from 'lucide-react'
import { memo, useCallback, useMemo, useState } from 'react'
import { create } from 'zustand'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { compact } from '../game/compact.ts'
import { NewProfileDialog } from '../game/NewProfileDialog.tsx'
import { ProfileContextMenu } from '../game/ProfileContextMenu.tsx'
import { useOrderedProfiles } from '../game/useSidebarProfiles.ts'
import { useGameName } from '../games/info.ts'
import { modsLabel } from '../i18n/counts.ts'
import { useNav } from '../nav/store.ts'
import { openImport } from '../share/store.ts'
import { MenuHeading, MenuRule, TitleMenu, TitleMenuItem } from '../shell/TitleMenu.tsx'
import { closeTitleMenu, useTitleMenu } from '../shell/titleMenus.ts'
import { visuallyHidden } from '../shell/visuallyHidden.ts'
import { ManageTemplatesDialog } from '../templates/TemplateDialogs.tsx'
import { useTemplates } from '../templates/useTemplates.ts'
import { space } from '../theme/density.ts'
import { userModCount } from './count.ts'
import { ProfileMark } from './ProfileMark.tsx'
import { openProfileOf, useProfiles } from './store.ts'

const ICON_PX = 18

interface Position {
  top: number
  left: number
}

function Dot({ mark }: { mark: Profile | undefined }) {
  if (mark && (mark.color || mark.icon)) {
    return <ProfileMark profile={mark} size={18} />
  }
  return (
    <Box
      component="span"
      aria-hidden={true}
      sx={{ width: 10, height: 10, flexShrink: 0, borderRadius: '50%', bgcolor: 'primary.main' }}
    />
  )
}

function ProfileButton({
  current,
  expanded,
  trigger,
  onContext,
}: {
  current: Profile | undefined
  expanded: boolean
  trigger: ReturnType<typeof useTitleMenu>['trigger']
  onContext: (id: string, position: Position) => void
}) {
  const { t } = useLingui()
  return (
    <ButtonBase
      data-tour="profile-switcher"
      aria-haspopup="menu"
      aria-expanded={expanded}
      {...trigger}
      onContextMenu={(e) => {
        if (current) {
          e.preventDefault()
          onContext(current.id, { top: e.clientY, left: e.clientX })
        }
      }}
      onKeyDown={(e) => {
        if (current && (e.key === 'ContextMenu' || (e.key === 'F10' && e.shiftKey))) {
          e.preventDefault()
          const rect = e.currentTarget.getBoundingClientRect()
          onContext(current.id, { top: rect.bottom, left: rect.left })
        }
      }}
      sx={{
        '--wails-draggable': 'no-drag',
        gap: '8px',
        height: space.control,
        minWidth: 0,
        px: '10px',
        borderRadius: '8px',
        fontFamily: 'inherit',
        fontSize: 'medium.fontSize',
        fontWeight: 600,
        color: 'inherit',
        bgcolor: 'var(--mortar-hairline-faint)',
        border: '1px solid var(--mortar-hairline-12)',
        '&:hover, &[aria-expanded="true"]': { bgcolor: 'var(--mortar-hairline-12)' },
      }}
    >
      <Dot mark={current} />
      {/* Named by its content, so the visible text is inside the accessible name. */}
      {current ? (
        <Box component="span" sx={visuallyHidden}>
          {`${t`Switch profile`}:`}
        </Box>
      ) : null}
      <Box
        component="span"
        sx={{
          minWidth: 0,
          overflow: 'hidden',
          textOverflow: 'ellipsis',
          whiteSpace: 'nowrap',
          [compact]: { maxWidth: 110 },
        }}
      >
        {current?.name ?? t`No profile`}
      </Box>
      {current ? (
        <Box
          component="span"
          sx={{
            flexShrink: 0,
            fontSize: 'small.fontSize',
            fontWeight: 400,
            px: '8px',
            py: '2px',
            borderRadius: '10px',
            whiteSpace: 'nowrap',
            color: 'var(--mortar-ink-90)',
            bgcolor: 'var(--mortar-hairline-faint)',
            [compact]: { display: 'none' },
          }}
        >
          {modsLabel(userModCount(current))}
        </Box>
      ) : null}
      <ChevronDown size={14} aria-hidden={true} style={{ flexShrink: 0 }} />
    </ButtonBase>
  )
}

interface MenuListProps {
  game: string
  gameName: string
  profiles: Profile[]
  openId: string
  hasTemplates: boolean
  onCreate: () => void
  onManage: () => void
  onContext: (id: string, position: Position) => void
}

function ProfileMenuListView({
  game,
  gameName,
  profiles,
  openId,
  hasTemplates,
  onCreate,
  onManage,
  onContext,
}: MenuListProps) {
  const { t } = useLingui()
  const profileRowLabel = (p: Profile) => {
    const { name } = p
    const mods = modsLabel(userModCount(p))
    return `${name} · ${mods}`
  }
  return (
    <>
      <MenuHeading>{t`${{ game: gameName }} profiles`}</MenuHeading>
      {profiles.map((p) => (
        <TitleMenuItem
          key={p.id}
          icon={<Dot mark={p} />}
          label={profileRowLabel(p)}
          checked={p.id === openId}
          onClick={() => {
            closeTitleMenu()
            useProfiles.getState().open(p.id)
            if (useNav.getState().route.name !== 'game') {
              useNav.getState().openGame(game)
            }
          }}
          onContextMenu={(e) => {
            e.preventDefault()
            closeTitleMenu()
            onContext(p.id, { top: e.clientY, left: e.clientX })
          }}
        />
      ))}
      <MenuRule />
      <TitleMenuItem
        icon={<Plus size={ICON_PX} />}
        label={t`New profile…`}
        onClick={() => {
          closeTitleMenu()
          onCreate()
        }}
      />
      <TitleMenuItem
        icon={<Download size={ICON_PX} />}
        label={t`Import…`}
        onClick={() => {
          closeTitleMenu()
          openImport()
        }}
      />
      <MenuRule />
      <TitleMenuItem
        icon={<ListOrdered size={ICON_PX} />}
        label={t`Manage profiles…`}
        onClick={() => {
          closeTitleMenu()
          useNav.getState().openProfiles()
        }}
      />
      {hasTemplates ? (
        <TitleMenuItem
          icon={<LayoutTemplate size={ICON_PX} />}
          label={t`Manage templates…`}
          onClick={() => {
            closeTitleMenu()
            onManage()
          }}
        />
      ) : null}
    </>
  )
}

// The menu's items, memoised: opening or closing the menu (and navigating, which closes it) re-renders the menu around
// them but not every item.
const ProfileMenuList = memo(
  ProfileMenuListView,
  (a, b) =>
    a.game === b.game &&
    a.gameName === b.gameName &&
    a.profiles === b.profiles &&
    a.openId === b.openId &&
    a.hasTemplates === b.hasTemplates &&
    a.onCreate === b.onCreate &&
    a.onManage === b.onManage &&
    a.onContext === b.onContext,
)

// Which profile's right-click menu is open, and where. Held outside the title-bar menu, so opening and closing it
// does not redraw that menu and its rows.
const useProfileContext = create<{ id: string; position: Position | null }>(() => ({
  id: '',
  position: null,
}))
const showContext = (id: string, position: Position) => useProfileContext.setState({ id, position })
const hideContext = () => useProfileContext.setState({ position: null })

// Memoised: the switcher opening and closing re-renders its view, and the right-click menu stays mounted under it.
const ContextHost = memo(function ContextHostView({ game }: { game: string }) {
  const id = useProfileContext((s) => s.id)
  const position = useProfileContext((s) => s.position)
  const profile = useProfiles((s) => s.profiles.find((p) => p.id === id))
  return profile ? (
    <ProfileContextMenu game={game} profile={profile} position={position} onClose={hideContext} />
  ) : null
})

// The title bar's profile switcher for the open game: the open profile and its mod count over a menu that switches,
// creates, imports and manages profiles. A profile's right-click menu opens from its row and from the button.
// Memoised: the title bar re-renders on every navigation, and this one does not need to.
export const ProfileMenu = memo(function ProfileMenuView({ game }: { game: string }) {
  const { t } = useLingui()
  const gameName = useGameName(game)
  const { profiles } = useOrderedProfiles(game)
  const openId = useProfiles((s) => s.openId)
  const current = useProfiles(openProfileOf)
  const { anchor, close, trigger } = useTitleMenu('profile')
  const [creating, setCreating] = useState(false)
  const [managing, setManaging] = useState(false)
  const { templates, reload } = useTemplates(game, anchor !== null || managing)
  const onCreate = useCallback(() => setCreating(true), [])
  const onManage = useCallback(() => setManaging(true), [])
  // Elements held between renders: a closed dialog is skipped when the profile list or open profile changes.
  const newProfile = useMemo(
    () => <NewProfileDialog open={creating} onClose={() => setCreating(false)} />,
    [creating],
  )
  const manageTemplates = useMemo(
    () => (
      <ManageTemplatesDialog
        open={managing}
        game={game}
        templates={templates}
        onChanged={reload}
        onClose={() => setManaging(false)}
      />
    ),
    [managing, game, templates, reload],
  )
  return (
    <>
      <ProfileButton
        current={current}
        expanded={anchor !== null}
        trigger={trigger}
        onContext={showContext}
      />
      <TitleMenu anchorEl={anchor} onClose={close} label={t`Profiles`} width={300}>
        <ProfileMenuList
          game={game}
          gameName={gameName}
          profiles={profiles}
          openId={openId}
          hasTemplates={templates.length > 0}
          onCreate={onCreate}
          onManage={onManage}
          onContext={showContext}
        />
      </TitleMenu>
      <ContextHost game={game} />
      {newProfile}
      {manageTemplates}
    </>
  )
})
