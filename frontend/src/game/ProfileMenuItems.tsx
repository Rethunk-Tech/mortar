import { useLingui } from '@lingui/react/macro'
import {
  Copy,
  FolderOpen,
  Gamepad2,
  GitCompare,
  History,
  ImageOff,
  ImagePlus,
  PackagePlus,
  ShieldCheck,
  Snowflake,
  SquareArrowOutUpRight,
  Star,
  Trash2,
} from 'lucide-react'
import { useEffect, useState } from 'react'
import { PickImage } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { SetFavourite } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import {
  AddToSteam,
  Create as CreateShortcut,
  Remove as RemoveShortcut,
  Capabilities as ShortcutCapabilities,
  Exists as ShortcutExists,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/shortcut/service.ts'
import { AddResult } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/steam/models.ts'
import { setFrozen } from '../mods/freeze.ts'
import { isFrozen } from '../mods/frozen.ts'
import { useProfiles } from '../profiles/store.ts'
import { MenuAction } from '../shell/MenuAction.tsx'
import { MenuRule } from '../shell/TitleMenu.tsx'
import { reportError, reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { BackupMenuItem, RestoreMenuItem } from './BackupMenuItems.tsx'
import { applyStagedCover, hasPickedCover } from './cover.ts'
import { FarmMenuItem } from './FarmMenuItem.tsx'
import { openProfileDialog } from './profileDialog.ts'
import { TemplateMenuItems } from './TemplateMenuItems.tsx'

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

// ShortcutMenuItems are the ways to start a profile from outside Mortar: a desktop or Start menu shortcut, and Steam.
function ShortcutMenuItems({ profile, close }: { profile: Profile; close: () => void }) {
  const { t } = useLingui()
  const currentGame = useProfiles((s) => s.game)
  const [hasShortcut, setHasShortcut] = useState(false)
  // Steam on the host cannot start Mortar inside a Flatpak, so the action is left out there.
  const [steamShortcut, setSteamShortcut] = useState(true)
  useEffect(() => {
    ShortcutCapabilities()
      .then((c) => setSteamShortcut(c.addToSteam))
      .catch(reportUnexpected)
  }, [])
  useEffect(() => {
    const g = currentGame
    if (!g) {
      setHasShortcut(false)
      return
    }
    ShortcutExists(g.id, profile.id)
      .then(setHasShortcut)
      .catch(() => setHasShortcut(false))
  }, [currentGame, profile.id])
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
              .then((path) =>
                useToasts
                  .getState()
                  .push({ kind: 'success', title: t`Shortcut added`, body: path }),
              )
              .catch(reportError(t`Could not add the shortcut`))
          }
        }}
      />
      {hasShortcut ? (
        <ProfileMenuItem
          icon={<Trash2 size={16} />}
          label={t`Remove the shortcut`}
          onClick={() => {
            close()
            const g = currentGame
            if (g) {
              RemoveShortcut(g.id, profile.id)
                .then(() => {
                  useToasts.getState().push({ kind: 'success', title: t`Shortcut removed` })
                  setHasShortcut(false)
                })
                .catch(reportError(t`Could not remove the shortcut`))
            }
          }}
        />
      ) : null}
      {steamShortcut ? (
        <ProfileMenuItem
          icon={<Gamepad2 size={16} />}
          label={t`Add this profile to Steam`}
          onClick={() => {
            close()
            const g = currentGame
            if (g) {
              AddToSteam(g.id, g.name, profile.id, profile.name)
                .then((result) =>
                  useToasts.getState().push(
                    result === AddResult.Unchanged
                      ? { kind: 'success', title: t`Already in Steam` }
                      : {
                          kind: 'success',
                          title:
                            result === AddResult.Updated ? t`Updated in Steam` : t`Added to Steam`,
                          body: t`It shows in your Steam library the next time Steam starts.`,
                        },
                  ),
                )
                .catch(reportError(t`Could not add it to Steam`))
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

function CoverMenuItems({
  game,
  profile,
  close,
}: {
  game: string
  profile: Profile
  close: () => void
}) {
  const { t } = useLingui()
  const apply = async (path: string | null) => {
    const next = await applyStagedCover(game, profile.id, path)
    if (next) {
      useProfiles.getState().replace(next)
    }
  }
  const choose = async () => {
    close()
    const path = await PickImage(t`Choose cover image`)
    if (path) {
      await apply(path).catch(reportError(t`Could not use that image`))
    }
  }
  return [
    <ProfileMenuItem
      key="choose-cover"
      icon={<ImagePlus size={16} />}
      label={t`Choose cover image…`}
      onClick={() => choose().catch(reportUnexpected)}
    />,
    <ProfileMenuItem
      key="automatic-cover"
      icon={<ImageOff size={16} />}
      label={t`Use the automatic cover`}
      disabled={!hasPickedCover(profile.cover, undefined)}
      onClick={() => {
        close()
        apply(null).catch(reportUnexpected)
      }}
    />,
  ]
}

// Sharing, exporting and backing up the profile, as one menu group.
function ShareMenuItems({ profile, close }: { profile: Profile; close: () => void }) {
  return [
    <FarmMenuItem key="farm" profile={profile} close={close} />,
    <BackupMenuItem key="backup" profile={profile} close={close} />,
    <RestoreMenuItem key="restore" close={close} />,
  ]
}

function MoreMenuItems({ profile, close }: { profile: Profile; close: () => void }) {
  const { t } = useLingui()
  const hasGame = useProfiles((s) => s.game !== null)
  const duplicate = useProfiles((s) => s.duplicate)
  const openId = useProfiles((s) => s.openId)
  return [
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
      key="folder"
      icon={<FolderOpen size={16} />}
      label={t`Profile folder…`}
      onClick={() => {
        close()
        openProfileDialog('folder', profile.id)
      }}
    />,
    <CheckProfileMenuItem key="check" profile={profile} close={close} />,
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
      {...(profile.id === openId ? {} : { tooltip: t`Open the profile first.` })}
      onClick={() => {
        close()
        setFrozen(!isFrozen(profile)).catch(reportUnexpected)
      }}
    />,
    <ProfileMenuItem
      key="duplicate"
      icon={<Copy size={16} />}
      label={t`Duplicate`}
      onClick={() => {
        close()
        duplicate(profile.id).catch(reportUnexpected)
      }}
    />,
    <MenuRule key="sharing-divider" />,
    <ProfileMenuItem
      key="compare"
      icon={<GitCompare size={16} />}
      label={t`Compare…`}
      onClick={() => {
        close()
        openProfileDialog('compare', profile.id)
      }}
    />,
    <ShareMenuItems key="share" profile={profile} close={close} />,
    <MenuRule key="saved-group-divider" />,
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
    <TemplateMenuItems key="templates" profile={profile} close={close} />,
    <MenuRule key="shortcut-divider" />,
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

export { CoverMenuItems, MoreMenuItems, ProfileMenuItem, ShortcutMenuItems }
