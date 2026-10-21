import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Alert, Box, Button } from '@mui/material'
import { Inbox } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import type { ProfilePreview } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/migrate/models.ts'
import { LocalProfiles } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/packsvc/service.ts'
import {
  ExternalPreview,
  ExternalVortexSupported,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import {
  Import,
  PreviewLocalProfile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/sharesvc/service.ts'
import { useExternalImportSources } from '../profiles/externalImportSources.ts'
import { FoundProfileRow } from '../profiles/FoundProfileRow.tsx'
import { PackFolderPrompt } from '../profiles/PackFolderPrompt.tsx'
import { VortexFolderPrompt } from '../profiles/VortexFolderPrompt.tsx'
import { offerVortexFolder } from '../profiles/vortexFolderOffer.ts'
import { EmptyState } from '../shell/EmptyState.tsx'
import { space } from '../theme/density.ts'
import { reportUnexpected, toastError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { showImported } from './useImportFlow.ts'

interface Found {
  key: string
  source: string
  name: string
  mods: number
  // An r2modman or Gale profile is read from its folder; another manager's by source kind and id.
  folder?: string
  external?: { kind: string; id: string }
  // Already brought into one of this game's profiles; importing it again would make a copy.
  imported?: boolean
}

// Each ticked r2modman or Gale profile becomes a new profile named after it, read the way a single pick is.
async function importEach(game: string, folders: string[], failed: string): Promise<string> {
  let first = ''
  let done = 0
  for (const folder of folders) {
    try {
      const preview = await PreviewLocalProfile(game, folder, '')
      const result = await Import(game, preview.session, '', [])
      first ||= result.profile.id
      done += 1
    } catch (error) {
      toastError(failed, error)
    }
  }
  if (done > 0) {
    useToasts.getState().push({
      kind: 'success',
      title: plural(done, { one: 'Imported # profile', other: 'Imported # profiles' }),
    })
  }
  return first
}

// The profiles other mod managers keep for this game on this computer: r2modman, Gale, Vortex, Mod Organizer 2 and
// Stardrop. Picking one previews it in the import dialog; ticking several r2modman or Gale profiles imports them all.
export function ManagerPane({
  game,
  onFolder,
  onExternal,
  onDone,
}: {
  game: string
  onFolder: (folder: string) => void
  onExternal: (preview: ProfilePreview) => void
  onDone: () => void
}) {
  const { t } = useLingui()
  const { sources: external, reload } = useExternalImportSources(game, true)
  const [vortexSupported, setVortexSupported] = useState(false)
  // null until this game's profiles are listed, so opening never flashes "No profiles found".
  const [folders, setFolders] = useState<Found[] | null>(null)
  const [chosen, setChosen] = useState<string[]>([])
  const [busy, setBusy] = useState(false)
  useEffect(() => {
    ExternalVortexSupported(game).then(setVortexSupported).catch(reportUnexpected)
  }, [game])
  const loadFolders = useCallback(() => {
    LocalProfiles(game)
      .then((found) =>
        setFolders(
          (found ?? []).map((p) => ({
            key: `folder:${p.path}`,
            source: p.source,
            name: p.name,
            mods: p.mods,
            folder: p.path,
            imported: p.imported,
          })),
        ),
      )
      .catch(reportUnexpected)
  }, [game])
  useEffect(loadFolders, [loadFolders])
  const all: Found[] = [
    ...(folders ?? []),
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
    if (f.folder) {
      onFolder(f.folder)
    } else if (f.external) {
      ExternalPreview(game, f.external.kind, f.external.id).then(onExternal).catch(reportUnexpected)
    }
  }
  const importChosen = async () => {
    setBusy(true)
    const first = await importEach(game, chosen, t`Could not import`)
    setBusy(false)
    onDone()
    // Like a single import, the batch leaves setup for the game, on the first profile it made.
    if (first !== '') {
      showImported(game, false, first).catch(reportUnexpected)
    }
  }
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5, p: '20px 18px' }}>
      {folders !== null && all.length === 0 ? (
        <EmptyState compact={true} icon={<Inbox size={28} />} title={t`No profiles found.`}>
          {t`Mortar found no other mod manager's profiles for this game on this computer.`}
        </EmptyState>
      ) : (
        all.map((f) => (
          <FoundProfileRow
            key={f.key}
            name={f.name}
            source={f.imported ? t`${f.source} · Imported` : f.source}
            mods={f.mods}
            disabled={busy || f.imported === true}
            {...(f.folder && !f.imported
              ? {
                  check: {
                    checked: chosen.includes(f.folder),
                    onChange: (on: boolean) =>
                      setChosen((c) =>
                        on ? [...c, f.folder as string] : c.filter((x) => x !== f.folder),
                      ),
                  },
                }
              : {})}
            onClick={() => pick(f)}
          />
        ))
      )}
      {external
        .filter((s) => s.error)
        .map((s) => (
          <Alert key={s.kind} severity="warning">
            {`${s.name}: ${s.error}`}
          </Alert>
        ))}
      <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap, mt: space.gap }}>
        <Box sx={{ mr: 'auto', display: 'flex', flexDirection: 'column' }}>
          <PackFolderPrompt onChosen={loadFolders} />
          {offerVortexFolder(vortexSupported, external) ? (
            <VortexFolderPrompt game={game} onChosen={reload} />
          ) : null}
        </Box>
        {chosen.length > 0 ? (
          <Button
            variant="contained"
            disabled={busy}
            onClick={() => importChosen().catch(reportUnexpected)}
          >
            {plural(chosen.length, { one: 'Import # profile', other: 'Import # profiles' })}
          </Button>
        ) : null}
      </Box>
    </Box>
  )
}
