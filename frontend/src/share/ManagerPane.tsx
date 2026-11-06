import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Alert, Box, Button } from '@mui/material'
import { Inbox } from 'lucide-react'
import { useCallback, useState } from 'react'
import type { ProfilePreview } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/migrate/models.ts'
import { LocalProfiles } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/packsvc/service.ts'
import {
  ExternalMO2Supported,
  ExternalPreview,
  ExternalVortexSupported,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import {
  Import,
  PreviewLocalProfile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/sharesvc/service.ts'
import { GALE, MOD_ORGANIZER_2, R2MODMAN } from '../brand/productNames.ts'
import { useExternalImportSources } from '../profiles/externalImportSources.ts'
import { FoundProfileRow } from '../profiles/FoundProfileRow.tsx'
import { ManagerFolderPrompt } from '../profiles/ManagerFolderPrompt.tsx'
import { VortexFolderPrompt } from '../profiles/VortexFolderPrompt.tsx'
import { offerMO2Folder, offerVortexFolder } from '../profiles/vortexFolderOffer.ts'
import { DialogFooter } from '../shell/dialog/AppDialog.tsx'
import { EmptyState } from '../shell/EmptyState.tsx'
import { RefreshFailed } from '../shell/RefreshFailed.tsx'
import { SkeletonRows } from '../shell/SkeletonRows.tsx'
import { useLastGood } from '../shell/useLastGood.ts'
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
      const result = await Import(game, preview.session, '', [], false)
      first ||= result.profile.id
      done += 1
    } catch (error) {
      toastError(failed, error)
    }
  }
  if (done > 0) {
    useToasts.getState().push({
      kind: 'success',
      title: plural({ count: done }, { one: 'Imported # profile', other: 'Imported # profiles' }),
    })
  }
  return first
}

// r2modman's and Gale's profiles by folder, then each other manager's by source and id.
function foundProfiles(
  folders: Found[] | null,
  external: ReturnType<typeof useExternalImportSources>['sources'],
): Found[] {
  return [
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
}

// The profiles other mod managers keep for this game on this computer: r2modman, Gale, Vortex, Mod Organizer 2 and
// Stardrop. Picking one previews it in the import dialog; ticking several r2modman or Gale profiles imports them all.
// The dialog places the body and the footer; nothing is read until active.
const toFound = (found: Awaited<ReturnType<typeof LocalProfiles>>): Found[] =>
  (found ?? []).map((p) => ({
    key: `folder:${p.path}`,
    source: p.source,
    name: p.name,
    mods: p.mods,
    folder: p.path,
    imported: p.imported,
  }))

interface Listed {
  folders: Found[]
  vortex: boolean
  mo2: boolean
}

// Read together, so the pane redraws once for all three answers.
const listFolders = async (game: string): Promise<Listed> => {
  const [found, vortex, mo2] = await Promise.all([
    LocalProfiles(game),
    ExternalVortexSupported(game),
    ExternalMO2Supported(game),
  ])
  return { folders: toFound(found), vortex, mo2 }
}

export function useManagerPane({
  game,
  active,
  onFolder,
  onExternal,
  onDone,
}: {
  game: string
  active: boolean
  onFolder: (folder: string) => void
  onExternal: (preview: ProfilePreview) => void
  onDone: () => void
}) {
  const { t } = useLingui()
  const { sources: external, reload } = useExternalImportSources(game, active)
  // null until this game's profiles were ever listed, so opening never flashes "No profiles found".
  const {
    data: listed,
    error: listError,
    at: listedAt,
    reload: relist,
  } = useLastGood<Listed>(
    `manager-pane:${game}`,
    active ? () => listFolders(game) : null,
    [active],
    { onError: reportUnexpected },
  )
  const folders = listed?.folders ?? null
  const vortexSupported = listed?.vortex ?? false
  const mo2Supported = listed?.mo2 ?? false
  const [chosen, setChosen] = useState<string[]>([])
  const [busy, setBusy] = useState(false)
  const loadFolders = useCallback(() => {
    relist().catch(reportUnexpected)
  }, [relist])
  const all = foundProfiles(folders, external)
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
  const body = (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5, px: 3, py: 2.5 }}>
      {folders === null ? null : (
        <RefreshFailed error={listError} at={listedAt} onRetry={loadFolders} />
      )}
      {folders === null ? (
        <SkeletonRows
          label={t`Looking for other mod managers' profiles…`}
          count={3}
          height={44}
          immediate={true}
        />
      ) : null}
      {folders !== null && all.length === 0 ? (
        <EmptyState compact={true} icon={<Inbox size={28} />} title={t`No profiles found`}>
          {t`Mortar found no other mod manager's profiles for this game on this computer.`}
        </EmptyState>
      ) : (
        all.map((f) => (
          <FoundProfileRow
            key={f.key}
            name={f.name}
            source={f.imported ? [f.source, t`Imported`].join(' · ') : f.source}
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
    </Box>
  )
  const footer = (
    <DialogFooter
      aside={
        <Box sx={{ display: 'flex', flexDirection: 'column' }}>
          <ManagerFolderPrompt
            settingKey="packFolder"
            label={t`${{ manager: R2MODMAN }} or ${{ otherManager: GALE }} data folder`}
            note={t`Use ${{ manager: R2MODMAN }} or ${{ otherManager: GALE }} from another folder?`}
            rejected={t`That folder has no ${{ manager: R2MODMAN }} or ${{ otherManager: GALE }} data. Choose ${{ otherManager: GALE }}'s data folder, the one holding data.sqlite3, or ${{ manager: R2MODMAN }}'s.`}
            onChosen={loadFolders}
          />
          {offerVortexFolder(vortexSupported, external) ? (
            <VortexFolderPrompt game={game} onChosen={reload} />
          ) : null}
          {offerMO2Folder(mo2Supported, external) ? (
            <ManagerFolderPrompt
              settingKey="mo2Folder"
              label={t`${{ name: MOD_ORGANIZER_2 }} folder`}
              note={t`Use ${{ manager: MOD_ORGANIZER_2 }} from another folder?`}
              rejected={t`That folder has no ${{ manager: MOD_ORGANIZER_2 }} data. Choose the one holding ModOrganizer.ini or ModOrganizer.exe.`}
              onChosen={reload}
            />
          ) : null}
        </Box>
      }
    >
      {chosen.length > 0 ? (
        <Button
          variant="contained"
          disabled={busy}
          onClick={() => importChosen().catch(reportUnexpected)}
        >
          {plural(
            { count: chosen.length },
            { one: 'Import # profile', other: 'Import # profiles' },
          )}
        </Button>
      ) : null}
    </DialogFooter>
  )
  return { active, body, footer, busy }
}
