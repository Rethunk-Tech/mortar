import { useLingui } from '@lingui/react/macro'
import { ListItemIcon, ListItemText, MenuItem } from '@mui/material'
import {
  FileDown,
  Gamepad2,
  History,
  ImageOff,
  ImagePlus,
  SquareArrowOutUpRight,
} from 'lucide-react'
import type { ReactNode } from 'react'
import { PickImage } from '../../bindings/github.com/Rethunk-AI/mortar/internal/picker/service.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  AddToSteam,
  Create as CreateShortcut,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/shortcut/service.ts'
import { useProfiles } from '../profiles/store.ts'
import { errorMessage, reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { applyStagedCover, hasPickedCover } from './cover.ts'

// The profile actions shared by the profile page's buttons and the sidebar's context menu, so both offer the same.

const toastError = (title: string) => (e: unknown) =>
  useToasts.getState().push({ kind: 'error', title, body: errorMessage(e) })

export function ProfileMenuItem({
  icon,
  label,
  disabled,
  onClick,
}: {
  icon: ReactNode
  label: string
  disabled?: boolean
  onClick: () => void
}) {
  return (
    <MenuItem disabled={disabled} onClick={onClick}>
      <ListItemIcon sx={{ color: 'inherit' }}>{icon}</ListItemIcon>
      <ListItemText>{label}</ListItemText>
    </MenuItem>
  )
}

export function CoverMenuItems({
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
      await apply(path).catch(toastError(t`Could not use that image`))
    }
  }
  return (
    <>
      <ProfileMenuItem
        icon={<ImagePlus size={16} />}
        label={t`Choose cover image…`}
        onClick={() => choose().catch(reportUnexpected)}
      />
      <ProfileMenuItem
        icon={<ImageOff size={16} />}
        label={t`Use the automatic cover`}
        disabled={!hasPickedCover(profile.cover, undefined)}
        onClick={() => {
          close()
          apply(null).catch(reportUnexpected)
        }}
      />
    </>
  )
}

export function MoreMenuItems({
  profile,
  close,
  onHistory,
}: {
  profile: Profile
  close: () => void
  onHistory: () => void
}) {
  const { t } = useLingui()
  const game = () => useProfiles.getState().game
  return (
    <>
      <ProfileMenuItem
        icon={<History size={16} />}
        label={t`History`}
        onClick={() => {
          close()
          onHistory()
        }}
      />
      <ProfileMenuItem
        icon={<FileDown size={16} />}
        label={t`Export profile…`}
        onClick={() => {
          close()
          useProfiles.getState().exportProfile(profile.id).catch(reportUnexpected)
        }}
      />
      <ProfileMenuItem
        icon={<SquareArrowOutUpRight size={16} />}
        label={t`Add a shortcut that plays this profile`}
        onClick={() => {
          close()
          const g = game()
          if (g) {
            CreateShortcut(g.id, g.name, profile.id, profile.name)
              .then((path) =>
                useToasts
                  .getState()
                  .push({ kind: 'success', title: t`Shortcut added`, body: path }),
              )
              .catch(toastError(t`Could not add the shortcut`))
          }
        }}
      />
      <ProfileMenuItem
        icon={<Gamepad2 size={16} />}
        label={t`Add this profile to Steam`}
        onClick={() => {
          close()
          const g = game()
          if (g) {
            AddToSteam(g.id, g.name, profile.id, profile.name)
              .then((added) =>
                useToasts.getState().push({
                  kind: 'success',
                  title: added ? t`Added to Steam` : t`Already in Steam`,
                  body: t`It shows in your Steam library the next time Steam starts.`,
                }),
              )
              .catch(toastError(t`Could not add it to Steam`))
          }
        }}
      />
    </>
  )
}
