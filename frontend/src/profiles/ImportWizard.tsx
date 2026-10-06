import { useLingui } from '@lingui/react/macro'
import { Button, Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material'
import { Inbox } from 'lucide-react'
import { useEffect, useState } from 'react'
import { LocalProfiles } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/packsvc/service.ts'
import { ExternalPreview } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { openImport } from '../share/store.ts'
import { EmptyState } from '../shell/EmptyState.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { useExternalImportSources } from './externalImportSources.ts'
import { FoundProfileRow } from './FoundProfileRow.tsx'

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
  // null until this game's manager profiles are listed, so opening never flashes "No profiles found".
  const [packs, setPacks] = useState<Found[] | null>(null)
  useEffect(() => {
    if (!open) {
      setPacks(null)
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
    ...(packs ?? []),
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
        {packs !== null && all.length === 0 ? (
          <EmptyState compact={true} icon={<Inbox size={28} />} title={t`No profiles found.`}>
            {t`Mortar found no other mod manager's profiles for this game on this computer.`}
          </EmptyState>
        ) : (
          all.map((f) => (
            <FoundProfileRow
              key={f.key}
              name={f.name}
              source={f.source}
              mods={f.mods}
              onClick={() => pick(f)}
            />
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
