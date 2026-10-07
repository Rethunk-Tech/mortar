import { useLingui } from '@lingui/react/macro'
import { Divider } from '@mui/material'
import {
  Copy,
  FileDown,
  Gamepad2,
  GitCompare,
  History,
  ImageOff,
  ImagePlus,
  PackagePlus,
  Send as SendIcon,
  ShieldCheck,
  SquareArrowOutUpRight,
  Trash2,
  Users,
} from 'lucide-react'
import { useEffect, useState } from 'react'
import { PickImage } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { SetLanSharing } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import {
  AddToSteam,
  Create as CreateShortcut,
  Remove as RemoveShortcut,
  Capabilities as ShortcutCapabilities,
  Exists as ShortcutExists,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/shortcut/service.ts'
import { AddResult } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/steam/models.ts'
import { bundleApplied } from '../bundles/applied.ts'
import { ApplyBundleDialog } from '../bundles/dialogs.tsx'
import { SendDialog } from '../lan/SendDialog.tsx'
import { CompareDialog } from '../profiles/CompareDialog.tsx'
import { HealthDialog } from '../profiles/HealthDialog.tsx'
import { useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import { openImport } from '../share/store.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { MenuAction } from '../shell/MenuAction.tsx'
import { reportError, reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { BackupMenuItems } from './BackupMenuItems.tsx'
import { applyStagedCover, hasPickedCover } from './cover.ts'
import { FarmMenuItem } from './FarmMenuItem.tsx'
import { TemplateMenuItems } from './TemplateMenuItems.tsx'

// The profile actions shared by the profile page's buttons and the sidebar's context menu, so both offer the same.

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
  const [open, setOpen] = useState(false)
  return (
    <>
      <ProfileMenuItem
        icon={<ShieldCheck size={16} />}
        label={t`Check this profile`}
        onClick={() => {
          close()
          setOpen(true)
        }}
      />
      <HealthDialog profileId={profile.id} open={open} onClose={() => setOpen(false)} />
    </>
  )
}

function SendProfileMenuItem({ profile, close }: { profile: Profile; close: () => void }) {
  const { t } = useLingui()
  const currentGame = useProfiles((s) => s.game)
  const lanSharing = useSettings((s) => s.lanSharing)
  const [open, setOpen] = useState(false)
  const [askSharing, setAskSharing] = useState(false)
  const [enabling, setEnabling] = useState(false)
  return (
    <>
      <ProfileMenuItem
        icon={<SendIcon size={16} />}
        label={t`Send to…`}
        disabled={!currentGame}
        onClick={() => {
          close()
          if (lanSharing) {
            setOpen(true)
          } else {
            setAskSharing(true)
          }
        }}
      />
      <ConfirmDialog
        open={askSharing}
        title={t`Turn on sharing nearby?`}
        body={t`Other Mortar users on your local network will be able to find this computer and send you profiles. You can turn it off again in Settings › General.`}
        confirmLabel={t`Turn on and continue`}
        busy={enabling}
        onCancel={() => setAskSharing(false)}
        onConfirm={() => {
          setEnabling(true)
          SetLanSharing(true)
            .then(() => {
              setAskSharing(false)
              setOpen(true)
            })
            .catch(reportError(t`Could not turn on sharing nearby`))
            .finally(() => setEnabling(false))
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
  compare,
  setCompare,
  deleting,
  setDeleting,
  bundleOpen,
  setBundleOpen,
}: {
  profile: Profile
  currentGame: { id: string } | null
  compare: { a: Profile; b: Profile } | null
  setCompare: (value: { a: Profile; b: Profile } | null) => void
  deleting: boolean
  setDeleting: (value: boolean) => void
  bundleOpen: boolean
  setBundleOpen: (value: boolean) => void
}) {
  return (
    <>
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
        onApplied={(result) => bundleApplied(result, profile.id)}
      />
    </>
  )
}

// Sharing, exporting and backing up the profile, as one menu group.
function ShareMenuItems({ profile, close }: { profile: Profile; close: () => void }) {
  return [
    <SendProfileMenuItem key="send" profile={profile} close={close} />,
    <FarmMenuItem key="farm" profile={profile} close={close} />,
    <BackupMenuItems key="backup" profile={profile} close={close} />,
  ]
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
    <CheckProfileMenuItem key="check" profile={profile} close={close} />,
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
        setCompare({ a: profile, b: profiles.find((p) => p.id !== profile.id) ?? profile })
      }}
    />,
    <ShareMenuItems key="share" profile={profile} close={close} />,
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
    <TemplateMenuItems key="templates" profile={profile} close={close} />,
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
      compare={compare}
      setCompare={setCompare}
      deleting={deleting}
      setDeleting={setDeleting}
      bundleOpen={bundleOpen}
      setBundleOpen={setBundleOpen}
    />,
  ]
}

export { CoverMenuItems, DeleteProfileDialog, MoreMenuItems, ProfileMenuItem, ShortcutMenuItems }
