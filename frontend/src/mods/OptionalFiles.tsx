import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
import type { KeyboardEvent } from 'react'
import type { File } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexus/models.ts'
import type {
  Mod,
  OverlayFileSet,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { OverlayFiles } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { useProfiles } from '../profiles/store.ts'
import { useLoaded } from '../shell/useLoaded.ts'
import { space } from '../theme/density.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { entryOf } from './lookup.ts'
import { useNexusEntry } from './nexusDetails.ts'
import { InstalledOptional, NexusOptional } from './OptionalFileRows.tsx'
import { alternativeGroups, installedFileIds, nexusOptionalFiles } from './optionalFiles.ts'
import { setOverlayEnabled } from './overlayActions.ts'
import { overlaysByBase } from './overlayRows.ts'
import { heading } from './paper.ts'
import type { OverlayRow } from './virtualRows.ts'

// What each optional file replaces and adds, read again whenever the profile changes.
const STEP: Partial<Record<string, number>> = {
  ArrowDown: 1,
  ArrowRight: 1,
  ArrowUp: -1,
  ArrowLeft: -1,
}

// Choosing a file swaps files on disk, so the arrows only move between a set's radios; Space or Enter chooses.
function moveBetweenRadios(e: KeyboardEvent<HTMLElement>) {
  const radios = [
    ...e.currentTarget.querySelectorAll<HTMLInputElement>('input[type="radio"]:not(:disabled)'),
  ]
  const at = e.target instanceof HTMLInputElement ? radios.indexOf(e.target) : -1
  if (at < 0) {
    return
  }
  if (e.key === 'Enter') {
    e.preventDefault()
    radios[at]?.click()
    return
  }
  const step = STEP[e.key]
  if (step === undefined) {
    return
  }
  e.preventDefault()
  radios[(at + step + radios.length) % radios.length]?.focus()
}

function useOverlaySets(profile: Profile, baseKey: string, count: number) {
  const game = useProfiles((s) => s.game?.id ?? '')
  const updated = String(profile.updated)
  const { data: sets } = useLoaded<OverlayFileSet[]>(
    game && baseKey && count > 0 && updated
      ? () => OverlayFiles(game, profile.id, baseKey).then((list) => list ?? [])
      : null,
    [game, profile.id, baseKey, count, updated],
    [],
    reportUnexpected,
  )
  return sets
}

/** The mod's Nexus Optional and Miscellaneous files: those in the profile first, then the rest of the page's. */
export function OptionalFiles({ mod, profile }: { mod: Mod; profile: Profile }) {
  const { t } = useLingui()
  const entry = entryOf(profile, mod.key)
  const nexusId = entry?.source.kind === 'nexus' ? (entry.source.modId ?? 0) : 0
  const files = useNexusEntry(nexusId)?.details?.files ?? []
  const rows = overlaysByBase(profile).get(mod.key) ?? []
  const sets = useOverlaySets(profile, mod.key, rows.length)
  const held = installedFileIds(profile, nexusId)
  const offered = nexusOptionalFiles(files).filter((f) => !held.has(f.fileId))
  if (rows.length === 0 && offered.length === 0) {
    return null
  }
  const groups = alternativeGroups(sets)
  const grouped = new Set(groups.flat())
  const fileOf = (row: OverlayRow): File | undefined => {
    const fileId = profile.entries?.find((e) => e.key === row.key)?.source.fileId ?? 0
    return files.find((f) => f.fileId === fileId)
  }
  const installed = (row: OverlayRow, radioName?: string) => (
    <InstalledOptional
      key={row.key}
      row={row}
      file={fileOf(row)}
      set={sets.find((s) => s.key === row.key)}
      radioName={radioName}
      onChoose={() => {
        setOverlayEnabled(row, true).catch(reportUnexpected)
      }}
    />
  )
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: space.gap }}>
      <Typography sx={heading}>{t`Optional files`}</Typography>
      {groups.map((group) => (
        <Box
          key={group.join('\n')}
          role="radiogroup"
          aria-label={t`Choose one`}
          onKeyDown={moveBetweenRadios}
          sx={{ display: 'flex', flexDirection: 'column', gap: 0.75 }}
        >
          <Typography sx={{ fontSize: 12, color: 'text.secondary' }}>{t`Choose one`}</Typography>
          {rows
            .filter((r) => group.includes(r.key))
            .map((r) => installed(r, `optional-${group.join('|')}`))}
        </Box>
      ))}
      {rows.filter((r) => !grouped.has(r.key)).map((r) => installed(r))}
      {offered.map((f) => (
        <NexusOptional
          key={f.fileId}
          file={f}
          modId={nexusId}
          modName={entry?.source.modName || mod.name}
        />
      ))}
    </Box>
  )
}
