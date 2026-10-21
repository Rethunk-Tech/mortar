import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase } from '@mui/material'
import { ChevronDown, LayoutTemplate, ListOrdered, Plus, Search } from 'lucide-react'
import { memo, useCallback, useMemo, useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { compact } from '../game/compact.ts'
import { NewProfileDialog } from '../game/NewProfileDialog.tsx'
import { ProfileContextMenu } from '../game/ProfileContextMenu.tsx'
import { useOrderedProfiles } from '../game/useSidebarProfiles.ts'
import { useGameName } from '../games/info.ts'
import { modsLabel } from '../i18n/counts.ts'
import { useNav } from '../nav/store.ts'
import { MenuHeading, MenuRule, TitleMenu, TitleMenuItem } from '../shell/TitleMenu.tsx'
import { closeTitleMenu, useTitleMenu } from '../shell/titleMenus.ts'
import { visuallyHidden } from '../shell/visuallyHidden.ts'
import { ManageTemplatesDialog } from '../templates/TemplateDialogs.tsx'
import { useTemplates } from '../templates/useTemplates.ts'
import { space } from '../theme/density.ts'
import { userModCount } from './count.ts'
import { requestFindAllFocus } from './findMod.ts'
import { ProfileMark } from './ProfileMark.tsx'
import { openProfileOf, useProfiles } from './store.ts'
import { type ImportEntry, useProfileImport } from './useProfileImport.tsx'

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
        fontSize: 14,
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
            fontSize: 12,
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
  entries: ImportEntry[]
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
  entries,
  hasTemplates,
  onCreate,
  onManage,
  onContext,
}: MenuListProps) {
  const { t } = useLingui()
  const profileRowLabel = (p: Profile) => {
    const { name } = p
    const mods = modsLabel(userModCount(p))
    return t`${name} · ${mods}`
  }
  return (
    <>
      <MenuHeading>{t`${gameName} profiles`}</MenuHeading>
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
      <MenuHeading>{t`Import`}</MenuHeading>
      {entries.map((entry) => (
        <TitleMenuItem
          key={entry.key}
          icon={<entry.icon size={ICON_PX} />}
          label={entry.label}
          onClick={() => {
            closeTitleMenu()
            entry.run()
          }}
        />
      ))}
      <MenuRule />
      <TitleMenuItem
        icon={<Search size={ICON_PX} />}
        label={t`Find a mod in all profiles…`}
        onClick={() => {
          closeTitleMenu()
          useNav.getState().openProfiles()
          requestFindAllFocus()
        }}
      />
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
// them but not every item. The import entries are rebuilt each render with the same actions, so only their keys count.
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
    a.onContext === b.onContext &&
    a.entries.map((e) => e.key).join() === b.entries.map((e) => e.key).join(),
)

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
  const [context, setContext] = useState<{ id: string; position: Position | null } | null>(null)
  const entries = useProfileImport()
  const { templates, reload } = useTemplates(game, anchor !== null || managing)
  const contextProfile = context ? profiles.find((p) => p.id === context.id) : undefined
  const showContext = useCallback(
    (id: string, position: Position) => setContext({ id, position }),
    [],
  )
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
          entries={entries}
          hasTemplates={templates.length > 0}
          onCreate={onCreate}
          onManage={onManage}
          onContext={showContext}
        />
      </TitleMenu>
      {contextProfile ? (
        <ProfileContextMenu
          game={game}
          profile={contextProfile}
          position={context?.position ?? null}
          onClose={() => setContext((c) => (c ? { ...c, position: null } : c))}
        />
      ) : null}
      {newProfile}
      {manageTemplates}
    </>
  )
})
