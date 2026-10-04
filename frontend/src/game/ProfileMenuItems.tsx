import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Divider } from '@mui/material'
import {
  Copy,
  Eye,
  EyeOff,
  FileDown,
  Gamepad2,
  GitCompare,
  History,
  ImageOff,
  ImagePlus,
  PackagePlus,
  Send as SendIcon,
  SquareArrowOutUpRight,
  Trash2,
  Users,
} from 'lucide-react'
import { useEffect, useState } from 'react'
import { PickImage } from '../../bindings/github.com/Rethunk-AI/mortar/internal/picker/service.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  AddToSteam,
  Create as CreateShortcut,
  Remove as RemoveShortcut,
  Exists as ShortcutExists,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/shortcut/service.ts'
import { ApplyBundleDialog } from '../bundles/dialogs.tsx'
import { SendDialog } from '../lan/SendDialog.tsx'
import { useMods } from '../mods/store.ts'
import { CompareDialog, PickCompareDialog } from '../profiles/CompareDialog.tsx'
import { useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import { openImport } from '../share/store.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { MenuAction } from '../shell/MenuAction.tsx'
import { errorMessage, reportError, reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { applyStagedCover, hasPickedCover } from './cover.ts'

// The profile actions shared by the profile page's buttons and the sidebar's context menu, so both offer the same.

const sentenceCase = (text: string) => text.charAt(0).toUpperCase() + text.slice(1)

function ProfileMenuItem({
  icon,
  label,
  disabled,
  tooltip,
  onClick,
}: {
  icon: React.ReactNode
  label: React.ReactNode
  disabled?: boolean
  tooltip?: string
  onClick: () => void
}) {
  return (
    <MenuAction icon={icon} label={label} disabled={disabled} tooltip={tooltip} onClick={onClick} />
  )
}

function DeleteProfileDialog({
  profile,
  open,
  onClose,
}: {
  profile: Profile
  open: boolean
  onClose: () => void
}) {
  const { t } = useLingui()
  const remove = useProfiles((s) => s.remove)
  return (
    <ConfirmDialog
      open={open}
      title={t`Delete ${profile.name}?`}
      body={t`The profile stays restorable for 30 days from Recently deleted.`}
      confirmLabel={t`Delete`}
      color="error"
      onCancel={onClose}
      onConfirm={() => {
        onClose()
        remove(profile.id).catch(reportUnexpected)
      }}
    />
  )
}

// ShortcutMenuItems are the ways to start a profile from outside Mortar: a desktop or Start menu shortcut, and Steam.
function ShortcutMenuItems({ profile, close }: { profile: Profile; close: () => void }) {
  const { t } = useLingui()
  const currentGame = useProfiles((s) => s.game)
  const [hasShortcut, setHasShortcut] = useState(false)
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
        label={t`Add a shortcut that plays this profile`}
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
      <ProfileMenuItem
        icon={<Gamepad2 size={16} />}
        label={t`Add this profile to Steam`}
        onClick={() => {
          close()
          const g = currentGame
          if (g) {
            AddToSteam(g.id, g.name, profile.id, profile.name)
              .then((added) =>
                useToasts.getState().push({
                  kind: 'success',
                  title: added ? t`Added to Steam` : t`Already in Steam`,
                  body: t`It shows in your Steam library the next time Steam starts.`,
                }),
              )
              .catch((e) =>
                useToasts.getState().push({
                  kind: 'error',
                  title: t`Could not add it to Steam`,
                  body: sentenceCase(errorMessage(e)),
                }),
              )
          }
        }}
      />
    </>
  )
}

function SendProfileMenuItem({ profile, close }: { profile: Profile; close: () => void }) {
  const { t } = useLingui()
  const currentGame = useProfiles((s) => s.game)
  const lanSharing = useSettings((s) => s.lanSharing)
  const [open, setOpen] = useState(false)
  return (
    <>
      <ProfileMenuItem
        icon={<SendIcon size={16} />}
        label={t`Send to…`}
        disabled={!(lanSharing && currentGame)}
        {...(lanSharing ? {} : { tooltip: t`Turn on sharing nearby in Settings › General.` })}
        onClick={() => {
          close()
          setOpen(true)
        }}
      />
      <SendDialog
        open={open}
        game={currentGame?.id ?? ''}
        profileId={profile.id}
        onClose={() => setOpen(false)}
      />
    </>
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

function ProfileDialogs({
  profile,
  currentGame,
  compareFrom,
  setCompareFrom,
  compare,
  setCompare,
  deleting,
  setDeleting,
  bundleOpen,
  setBundleOpen,
}: {
  profile: Profile
  currentGame: { id: string } | null
  compareFrom: Profile | null
  setCompareFrom: (profile: Profile | null) => void
  compare: { a: Profile; b: Profile } | null
  setCompare: (value: { a: Profile; b: Profile } | null) => void
  deleting: boolean
  setDeleting: (value: boolean) => void
  bundleOpen: boolean
  setBundleOpen: (value: boolean) => void
}) {
  const { t } = useLingui()
  return (
    <>
      {compareFrom === null ? null : (
        <PickCompareDialog
          from={compareFrom}
          onPicked={(other) => {
            setCompare({ a: compareFrom, b: other })
            setCompareFrom(null)
          }}
          onClose={() => setCompareFrom(null)}
        />
      )}
      <CompareDialog
        a={compare?.a ?? null}
        b={compare?.b ?? null}
        onClose={() => setCompare(null)}
      />
      <DeleteProfileDialog profile={profile} open={deleting} onClose={() => setDeleting(false)} />
      <ApplyBundleDialog
        open={bundleOpen}
        game={currentGame?.id ?? ''}
        profileName={profile.name}
        profileId={profile.id}
        onClose={() => setBundleOpen(false)}
        onApplied={(result) => {
          useProfiles.getState().replace(result.profile)
          if (useProfiles.getState().openId === profile.id) {
            useMods.getState().load().catch(reportUnexpected)
          }
          const missing = result.missing ?? []
          const added = plural(result.added, { one: '# mod added', other: '# mods added' })
          useToasts.getState().push({
            kind: 'success',
            title: t`Bundle added`,
            body:
              missing.length > 0
                ? `${added}\n${t`Not in Mortar's store: ${missing.join(', ')}`}`
                : added,
          })
        }}
      />
    </>
  )
}

function MoreMenuItems({
  profile,
  close,
  onHistory,
}: {
  profile: Profile
  close: () => void
  onHistory: () => void
}) {
  const { t } = useLingui()
  const profiles = useProfiles((s) => s.profiles)
  const currentGame = useProfiles((s) => s.game)
  const duplicate = useProfiles((s) => s.duplicate)
  const setHidden = useProfiles((s) => s.setHidden)
  const [compareFrom, setCompareFrom] = useState<Profile | null>(null)
  const [compare, setCompare] = useState<{ a: Profile; b: Profile } | null>(null)
  const [deleting, setDeleting] = useState(false)
  const [bundleOpen, setBundleOpen] = useState(false)
  return [
    <ProfileMenuItem
      key="history"
      icon={<History size={16} />}
      label={t`History`}
      onClick={() => {
        close()
        onHistory()
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
    <ProfileMenuItem
      key="export"
      icon={<FileDown size={16} />}
      label={t`Export profile…`}
      onClick={() => {
        close()
        useProfiles.getState().exportProfile(profile.id).catch(reportUnexpected)
      }}
    />,
    <ProfileMenuItem
      key="hide"
      icon={profile.hidden ? <Eye size={16} /> : <EyeOff size={16} />}
      label={profile.hidden ? t`Show in sidebar` : t`Hide from sidebar`}
      onClick={() => {
        close()
        setHidden(profile.id, !profile.hidden).catch(reportUnexpected)
      }}
    />,
    <Divider key="sharing-divider" />,
    <ProfileMenuItem
      key="match"
      icon={<Users size={16} />}
      label={t`Match a friend's profile…`}
      onClick={() => {
        close()
        openImport({ profileId: profile.id })
      }}
    />,
    <ProfileMenuItem
      key="compare"
      icon={<GitCompare size={16} />}
      label={t`Compare with…`}
      disabled={profiles.length < 2}
      onClick={() => {
        close()
        setCompareFrom(profile)
      }}
    />,
    <SendProfileMenuItem key="send" profile={profile} close={close} />,
    <Divider key="bundle-divider" />,
    <ProfileMenuItem
      key="bundle"
      icon={<PackagePlus size={16} />}
      label={t`Add a bundle…`}
      disabled={!currentGame}
      onClick={() => {
        close()
        setBundleOpen(true)
      }}
    />,
    <Divider key="shortcut-divider" />,
    <ShortcutMenuItems key="shortcuts" profile={profile} close={close} />,
    <Divider key="delete-divider" />,
    <ProfileMenuItem
      key="delete"
      icon={<Trash2 size={16} />}
      label={t`Delete`}
      onClick={() => {
        close()
        setDeleting(true)
      }}
    />,
    <ProfileDialogs
      key="dialogs"
      profile={profile}
      currentGame={currentGame}
      compareFrom={compareFrom}
      setCompareFrom={setCompareFrom}
      compare={compare}
      setCompare={setCompare}
      deleting={deleting}
      setDeleting={setDeleting}
      bundleOpen={bundleOpen}
      setBundleOpen={setBundleOpen}
    />,
  ]
}

export { CoverMenuItems, MoreMenuItems, ProfileMenuItem }
