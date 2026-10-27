import { useLingui } from '@lingui/react/macro'
import {
  Copy,
  FolderInput,
  FolderOpen,
  Gamepad2,
  GitCompare,
  GitMerge,
  History,
  PackagePlus,
  ShieldCheck,
  Snowflake,
  SquareArrowOutUpRight,
  Star,
  Trash2,
} from 'lucide-react'
import { useEffect, useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { SetFavourite } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import {
  AddToSteam,
  Create as CreateShortcut,
  Remove as RemoveShortcut,
  Capabilities as ShortcutCapabilities,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/shortcut/service.ts'
import { AddResult } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/steam/models.ts'
import { STEAM } from '../brand/productNames.ts'
import { setFrozen } from '../mods/freeze.ts'
import { isFrozen } from '../mods/frozen.ts'
import { useProfiles } from '../profiles/store.ts'
import { MenuAction } from '../shell/MenuAction.tsx'
import { MenuHeading, MenuRule } from '../shell/TitleMenu.tsx'
import { reportError, reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { BackupMenuItem, RestoreMenuItem } from './BackupMenuItems.tsx'
import { FarmMenuItem } from './FarmMenuItem.tsx'
import { openProfileDialog } from './profileDialog.ts'
import { lookForShortcut, setShortcut, useHasShortcut } from './shortcutLooks.ts'
import {
  ApplyTemplateMenuItem,
  ManageTemplatesMenuItem,
  SaveTemplateMenuItem,
} from './TemplateMenuItems.tsx'

// The profile actions shared by the profile page's buttons and the sidebar's context menu, so both offer the same.

function ProfileMenuItem({
  icon,
  label,
  disabled,
  tooltip,
  tone,
  onClick,
}: {
  icon: React.ReactNode
  label: React.ReactNode
  disabled?: boolean
  tooltip?: string
  tone?: 'error'
  onClick: () => void
}) {
  return (
    <MenuAction
      icon={icon}
      label={label}
      disabled={disabled}
      reason={tooltip}
      tone={tone}
      onClick={onClick}
    />
  )
}

// What shortcuts this install can make is fixed for the run, so it is asked once.
let capabilities: ReturnType<typeof ShortcutCapabilities> | null = null
const shortcutCapabilities = () => {
  capabilities ??= ShortcutCapabilities()
  return capabilities
}

// ShortcutMenuItems are the ways to start a profile from outside Mortar: a desktop or Start menu shortcut, and Steam.
function ShortcutMenuItems({ profile, close }: { profile: Profile; close: () => void }) {
  const { t } = useLingui()
  const currentGame = useProfiles((s) => s.game)
  const gameId = currentGame?.id ?? ''
  const hasShortcut = useHasShortcut(gameId, profile.id)
  useEffect(() => {
    if (gameId) {
      lookForShortcut(gameId, profile.id)
    }
  }, [gameId, profile.id])
  // Steam on the host cannot start Mortar inside a Flatpak, so the action is left out there.
  const [steamShortcut, setSteamShortcut] = useState(true)
  useEffect(() => {
    shortcutCapabilities()
      .then((c) => setSteamShortcut(c.addToSteam))
      .catch(reportUnexpected)
  }, [])
  return (
    <>
      <ProfileMenuItem
        icon={<SquareArrowOutUpRight size={16} />}
        label={t`Add a desktop shortcut`}
        onClick={() => {
          close()
          const g = currentGame
          if (g) {
            CreateShortcut(g.id, g.name, profile.id, profile.name)
              .then((path) => {
                setShortcut(g.id, profile.id, true)
                useToasts.getState().push({ kind: 'success', title: t`Shortcut added`, body: path })
              })
              .catch(reportError(t`Could not add the shortcut`))
          }
        }}
      />
      {hasShortcut ? (
        <ProfileMenuItem
          icon={<Trash2 size={16} />}
          label={t`Remove the shortcut`}
          tone="error"
          onClick={() => {
            close()
            const g = currentGame
            if (g) {
              RemoveShortcut(g.id, profile.id)
                .then(() => {
                  useToasts.getState().push({ kind: 'success', title: t`Shortcut removed` })
                  setShortcut(g.id, profile.id, false)
                })
                .catch(reportError(t`Could not remove the shortcut`))
            }
          }}
        />
      ) : null}
      {steamShortcut ? (
        <ProfileMenuItem
          icon={<Gamepad2 size={16} />}
          label={t`Add this profile to ${{ launcher: STEAM }}`}
          onClick={() => {
            close()
            const g = currentGame
            if (g) {
              AddToSteam(g.id, g.name, profile.id, profile.name)
                .then((result) =>
                  useToasts.getState().push(
                    result === AddResult.Unchanged
                      ? { kind: 'success', title: t`Already in ${{ launcher: STEAM }}` }
                      : {
                          kind: 'success',
                          title:
                            result === AddResult.Updated
                              ? t`Updated in ${{ launcher: STEAM }}`
                              : t`Added to ${{ launcher: STEAM }}`,
                          body: t`It shows in your ${{ launcher: STEAM }} library the next time ${{ launcher: STEAM }} starts.`,
                        },
                  ),
                )
                .catch(reportError(t`Could not add it to ${{ launcher: STEAM }}`))
            }
          }}
        />
      ) : null}
    </>
  )
}

function CheckProfileMenuItem({ profile, close }: { profile: Profile; close: () => void }) {
  const { t } = useLingui()
  return (
    <ProfileMenuItem
      icon={<ShieldCheck size={16} />}
      label={t`Check this profile`}
      onClick={() => {
        close()
        openProfileDialog('health', profile.id)
      }}
    />
  )
}

type Item = { profile: Profile; close: () => void }

// History, comparison and the health check: looking at the profile without changing it.
function ReviewItems({ profile, close }: Item) {
  const { t } = useLingui()
  return [
    <MenuRule key="review-divider" />,
    <MenuHeading key="review-heading">{t`Review`}</MenuHeading>,
    <ProfileMenuItem
      key="history"
      icon={<History size={16} />}
      label={t`History…`}
      onClick={() => {
        close()
        openProfileDialog('history', profile.id)
      }}
    />,
    <ProfileMenuItem
      key="compare"
      icon={<GitCompare size={16} />}
      label={t`Compare…`}
      onClick={() => {
        close()
        openProfileDialog('compare', profile.id)
      }}
    />,
    <CheckProfileMenuItem key="check" profile={profile} close={close} />,
  ]
}

// What makes another profile, template or list from this one, then the backup file and its restore.
function CopyItems({ profile, close }: Item) {
  const { t } = useLingui()
  const duplicate = useProfiles((s) => s.duplicate)
  const others = useProfiles((s) => s.profiles.length > 1)
  return [
    <MenuRule key="copies-divider" />,
    <MenuHeading key="copies-heading">{t`Copies`}</MenuHeading>,
    <ProfileMenuItem
      key="duplicate"
      icon={<Copy size={16} />}
      label={t`Duplicate`}
      onClick={() => {
        close()
        duplicate(profile.id).catch(reportUnexpected)
      }}
    />,
    <ProfileMenuItem
      key="merge"
      icon={<GitMerge size={16} />}
      label={t`Add these mods to another profile…`}
      disabled={!others}
      {...(others ? {} : { tooltip: t`No other profile to add them to.` })}
      onClick={() => {
        close()
        openProfileDialog('merge', profile.id)
      }}
    />,
    <SaveTemplateMenuItem key="save-template" profile={profile} close={close} />,
    <ManageTemplatesMenuItem key="manage-templates" profile={profile} close={close} />,
    <FarmMenuItem key="farm" profile={profile} close={close} />,
    <MenuRule key="backup-divider" />,
    <MenuHeading key="backup-heading">{t`Backup`}</MenuHeading>,
    <BackupMenuItem key="backup" profile={profile} close={close} />,
    <RestoreMenuItem key="restore" close={close} />,
  ]
}

// The profile's whole menu after Rename and Edit profile…, grouped as Home's Profile menu is.
function MoreMenuItems({ profile, close }: Item) {
  const { t } = useLingui()
  const hasGame = useProfiles((s) => s.game !== null)
  const openId = useProfiles((s) => s.openId)
  const openFolder = useProfiles((s) => s.openFolder)
  return [
    <ProfileMenuItem
      key="favourite"
      icon={<Star size={16} fill={profile.favourite ? 'currentColor' : 'none'} />}
      label={profile.favourite ? t`Remove from favourites` : t`Add to favourites`}
      onClick={() => {
        close()
        const game = useProfiles.getState().game?.id
        if (game) {
          SetFavourite(game, profile.id, !profile.favourite)
            .then((next) => useProfiles.getState().replace(next))
            .catch(reportUnexpected)
        }
      }}
    />,
    <ProfileMenuItem
      key="freeze"
      icon={<Snowflake size={16} />}
      label={isFrozen(profile) ? t`Unfreeze` : t`Freeze this profile`}
      disabled={profile.id !== openId}
      {...(profile.id === openId ? {} : { tooltip: t`Open profile first.` })}
      onClick={() => {
        close()
        setFrozen(!isFrozen(profile)).catch(reportUnexpected)
      }}
    />,
    <ProfileMenuItem
      key="open-folder"
      icon={<FolderOpen size={16} />}
      label={t`Open profile folder`}
      onClick={() => {
        close()
        openFolder(profile.id).catch(reportUnexpected)
      }}
    />,
    <ProfileMenuItem
      key="folder"
      icon={<FolderInput size={16} />}
      label={t`Move profile folder…`}
      onClick={() => {
        close()
        openProfileDialog('folder', profile.id)
      }}
    />,
    <ReviewItems key="review" profile={profile} close={close} />,
    <MenuRule key="add-divider" />,
    <MenuHeading key="add-heading">{t`Add mods`}</MenuHeading>,
    <ProfileMenuItem
      key="saved-group"
      icon={<PackagePlus size={16} />}
      label={t`Add a saved group…`}
      disabled={!hasGame}
      onClick={() => {
        close()
        openProfileDialog('savedGroup', profile.id)
      }}
    />,
    <ApplyTemplateMenuItem key="apply-template" profile={profile} close={close} />,
    <CopyItems key="copies" profile={profile} close={close} />,
    <MenuRule key="shortcuts-divider" />,
    <MenuHeading key="shortcuts-heading">{t`Shortcuts`}</MenuHeading>,
    <ShortcutMenuItems key="shortcuts" profile={profile} close={close} />,
    <MenuRule key="delete-divider" />,
    <ProfileMenuItem
      key="delete"
      icon={<Trash2 size={16} />}
      label={t`Delete profile…`}
      tone="error"
      onClick={() => {
        close()
        openProfileDialog('delete', profile.id)
      }}
    />,
  ]
}

export { MoreMenuItems, ProfileMenuItem, ShortcutMenuItems }
