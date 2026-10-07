import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase } from '@mui/material'
import { ChevronDown } from 'lucide-react'
import { useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { compact } from '../game/compact.ts'
import { NewProfileDialog } from '../game/NewProfileDialog.tsx'
import { ProfileContextMenu } from '../game/ProfileContextMenu.tsx'
import { useOrderedProfiles } from '../game/useSidebarProfiles.ts'
import { useGameName } from '../games/info.ts'
import { modsLabel } from '../i18n/counts.ts'
import { useNav } from '../nav/store.ts'
import { MenuHeading, MenuRule, TitleMenu, TitleMenuItem } from '../shell/TitleMenu.tsx'
import { useTitleMenu } from '../shell/titleMenus.ts'
import { ManageTemplatesDialog } from '../templates/TemplateDialogs.tsx'
import { useTemplates } from '../templates/useTemplates.ts'
import { userModCount } from './count.ts'
import { requestFindAllFocus } from './findMod.ts'
import { ProfileMark } from './ProfileMark.tsx'
import { openProfileOf, useProfiles } from './store.ts'
import { useProfileImport } from './useProfileImport.tsx'

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
  const mods = current ? modsLabel(userModCount(current)) : ''
  return (
    <ButtonBase
      aria-label={current ? t`Switch profile: ${current.name} ${mods}` : t`No profile`}
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
        height: 34,
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

// The title bar's profile switcher for the open game: the open profile and its mod count over a menu that switches,
// creates, imports and manages profiles. A profile's right-click menu opens from its row and from the button.
export function ProfileMenu({ game }: { game: string }) {
  const { t } = useLingui()
  const gameName = useGameName(game)
  const { profiles } = useOrderedProfiles(game)
  const openId = useProfiles((s) => s.openId)
  const open = useProfiles((s) => s.open)
  const current = useProfiles(openProfileOf)
  const { anchor, close, trigger } = useTitleMenu('profile')
  const [creating, setCreating] = useState(false)
  const [managing, setManaging] = useState(false)
  const [context, setContext] = useState<{ id: string; position: Position | null } | null>(null)
  const { entries, dialogs } = useProfileImport(game)
  const { templates, reload } = useTemplates(game, anchor !== null || managing)
  const contextProfile = context ? profiles.find((p) => p.id === context.id) : undefined
  const profileRowLabel = (p: Profile) => {
    const { name } = p
    const mods = modsLabel(userModCount(p))
    return t`${name} · ${mods}`
  }
  const showContext = (id: string, position: Position) => setContext({ id, position })
  return (
    <>
      <ProfileButton
        current={current}
        expanded={anchor !== null}
        trigger={trigger}
        onContext={showContext}
      />
      <TitleMenu anchorEl={anchor} onClose={close} label={t`Profiles`} width={300}>
        <MenuHeading>{t`${gameName} profiles`}</MenuHeading>
        {profiles.map((p) => (
          <TitleMenuItem
            key={p.id}
            label={profileRowLabel(p)}
            checked={p.id === openId}
            onClick={() => {
              close()
              open(p.id)
              if (useNav.getState().route.name !== 'game') {
                useNav.getState().openGame(game)
              }
            }}
            onContextMenu={(e) => {
              e.preventDefault()
              close()
              showContext(p.id, { top: e.clientY, left: e.clientX })
            }}
          />
        ))}
        <MenuRule />
        <TitleMenuItem
          label={t`New profile…`}
          onClick={() => {
            close()
            setCreating(true)
          }}
        />
        <MenuHeading>{t`Import`}</MenuHeading>
        {entries.map((entry) => (
          <TitleMenuItem
            key={entry.key}
            label={entry.label}
            onClick={() => {
              close()
              entry.run()
            }}
          />
        ))}
        <MenuRule />
        <TitleMenuItem
          label={t`Find a mod in all profiles…`}
          onClick={() => {
            close()
            useNav.getState().openProfiles()
            requestFindAllFocus()
          }}
        />
        <TitleMenuItem
          label={t`Manage profiles…`}
          onClick={() => {
            close()
            useNav.getState().openProfiles()
          }}
        />
        {templates.length > 0 ? (
          <TitleMenuItem
            label={t`Manage templates…`}
            onClick={() => {
              close()
              setManaging(true)
            }}
          />
        ) : null}
      </TitleMenu>
      {contextProfile ? (
        <ProfileContextMenu
          game={game}
          profile={contextProfile}
          position={context?.position ?? null}
          onClose={() => setContext((c) => (c ? { ...c, position: null } : c))}
        />
      ) : null}
      <NewProfileDialog open={creating} onClose={() => setCreating(false)} />
      <ManageTemplatesDialog
        open={managing}
        game={game}
        templates={templates}
        onChanged={reload}
        onClose={() => setManaging(false)}
      />
      {dialogs}
    </>
  )
}
