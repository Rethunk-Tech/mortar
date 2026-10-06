import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Typography,
} from '@mui/material'
import { Inbox } from 'lucide-react'
import { useEffect, useState } from 'react'
import { LocalProfiles } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/packsvc/service.ts'
import { ExternalPreview } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { modsLabel } from '../i18n/counts.ts'
import { openImport } from '../share/store.ts'
import { EmptyState } from '../shell/EmptyState.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { useExternalImportSources } from './externalImportSources.ts'

interface Found {
  key: string
  source: string
  name: string
  mods: number
  // A pack profile reads from a folder; an external one is previewed by source kind and id.
  pack?: string
  external?: { kind: string; id: string }
}

// One place to import from every manager found on this computer: r2modman, Gale, Vortex, Mod Organizer 2 and
// Stardrop. Picking a profile hands it to the same import path its manager always used.
export function ImportWizard({
  open,
  game,
  onClose,
  onPickPack,
  onOwnCode,
}: {
  open: boolean
  game: string
  onClose: () => void
  onPickPack: (path: string) => void
  // Shown for games with Thunderstore packs: a profile code or file instead of a profile on this computer.
  onOwnCode: (() => void) | null
}) {
  const { t } = useLingui()
  const external = useExternalImportSources(game)
  const [packs, setPacks] = useState<Found[]>([])
  useEffect(() => {
    if (!open) {
      return
    }
    LocalProfiles(game)
      .then((found) =>
        setPacks(
          (found ?? []).map((p) => ({
            key: `pack:${p.path}`,
            source: p.source,
            name: p.name,
            mods: p.mods,
            pack: p.path,
          })),
        ),
      )
      .catch(reportUnexpected)
  }, [open, game])
  const all: Found[] = [
    ...packs,
    ...external.flatMap((s) =>
      (s.profiles ?? []).map((p) => ({
        key: `${s.kind}:${p.id}`,
        source: s.name,
        name: p.name,
        mods: p.mods,
        external: { kind: s.kind, id: p.id },
      })),
    ),
  ]
  const pick = (f: Found) => {
    if (f.pack) {
      onClose()
      onPickPack(f.pack)
      return
    }
    if (f.external) {
      ExternalPreview(game, f.external.kind, f.external.id)
        .then((preview) => {
          openImport({ external: preview })
          onClose()
        })
        .catch(reportUnexpected)
    }
  }
  return (
    <Dialog open={open} onClose={onClose} fullWidth={true} maxWidth="sm">
      <DialogTitle>{t`Import a profile`}</DialogTitle>
      <DialogContent dividers={true} sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
        {all.length === 0 ? (
          <EmptyState compact={true} icon={<Inbox size={28} />} title={t`No profiles found.`}>
            {t`Mortar looked for r2modman, Gale, Vortex, Mod Organizer 2 and Stardrop profiles for this game on this computer.`}
          </EmptyState>
        ) : (
          all.map((f) => (
            <Button
              key={f.key}
              onClick={() => pick(f)}
              sx={{ justifyContent: 'space-between', textTransform: 'none', gap: 2 }}
            >
              <Box sx={{ textAlign: 'left', minWidth: 0 }}>
                <Typography title={f.name} noWrap={true} sx={{ fontWeight: 600 }}>
                  {f.name}
                </Typography>
                <Typography color="text.secondary" sx={{ fontSize: 12 }}>
                  {f.source}
                </Typography>
              </Box>
              <Typography color="text.secondary" sx={{ fontSize: 13, flexShrink: 0 }}>
                {modsLabel(f.mods)}
              </Typography>
            </Button>
          ))
        )}
      </DialogContent>
      <DialogActions>
        {onOwnCode ? (
          <Button
            onClick={() => {
              onClose()
              onOwnCode()
            }}
          >
            {t`Use a code or file…`}
          </Button>
        ) : null}
        <Button onClick={onClose}>{t`Cancel`}</Button>
      </DialogActions>
    </Dialog>
  )
}
