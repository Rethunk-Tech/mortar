import { useLingui } from '@lingui/react/macro'
import { Box, Collapse, IconButton, Typography } from '@mui/material'
import { ChevronDown, ChevronRight, Inbox } from 'lucide-react'
import { useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { EmptyState } from '../shell/EmptyState.tsx'
import { SearchField } from '../shell/SearchField.tsx'
import { CompareDiffRow, CompareSection } from './CompareRows.tsx'
import { OnlyInSection, VersionSection } from './CompareSections.tsx'
import type { ComparePair, CompareSide, ProfileCompare, SectionShared } from './compare.ts'
import { sideLabel } from './compare.ts'

function IdenticalList({ rows }: { rows: ComparePair[] }) {
  const { t } = useLingui()
  const [open, setOpen] = useState(false)
  if (rows.length === 0) {
    return null
  }
  return (
    <Box>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
        <IconButton
          size="small"
          aria-expanded={open}
          aria-label={t`Identical mods`}
          onClick={() => setOpen((value) => !value)}
        >
          {open ? <ChevronDown size={18} /> : <ChevronRight size={18} />}
        </IconButton>
        <Typography sx={{ fontSize: 13, fontWeight: 700, color: 'text.secondary' }}>
          {t`Identical (${rows.length})`}
        </Typography>
      </Box>
      <Collapse in={open}>
        <Box sx={{ pl: 4, pt: 0.5 }}>
          {rows.map((row) => (
            <Typography key={row.uniqueId} sx={{ fontSize: 14, py: 0.25 }}>
              {t`${row.name} · ${row.a.version}`}
            </Typography>
          ))}
        </Box>
      </Collapse>
    </Box>
  )
}

function CompareModFilter({
  value,
  onChange,
}: {
  value: string
  onChange: (value: string) => void
}) {
  const { t } = useLingui()
  return (
    <SearchField
      label={t`Filter mods`}
      fullWidth={true}
      value={value}
      onChange={onChange}
      sx={{ mb: 2 }}
    />
  )
}

function filteredCompare(diff: ProfileCompare, needle: string) {
  const matchesSide = (side: CompareSide) =>
    !needle ||
    side.name.toLowerCase().includes(needle) ||
    side.uniqueId.toLowerCase().includes(needle)
  const matchesPair = (row: ComparePair) =>
    !needle ||
    row.name.toLowerCase().includes(needle) ||
    row.uniqueId.toLowerCase().includes(needle)
  const onlyA = diff.onlyA.filter(matchesSide)
  const onlyB = diff.onlyB.filter(matchesSide)
  const differentVersion = diff.differentVersion.filter(matchesPair)
  const differentEnabled = diff.differentEnabled.filter(matchesPair)
  const identical = diff.identical.filter(matchesPair)
  const hasDiff =
    onlyA.length > 0 ||
    onlyB.length > 0 ||
    differentVersion.length > 0 ||
    differentEnabled.length > 0
  return { onlyA, onlyB, differentVersion, differentEnabled, identical, hasDiff }
}

export function CompareBulkBody({
  diff,
  profileA,
  profileB,
  aName,
  bName,
  pending,
  lockedReason,
  onCopy,
  onMatch,
}: {
  diff: ProfileCompare
  profileA: Profile
  profileB: Profile
  aName: string
  bName: string
  pending: boolean
  lockedReason: (profile: Profile) => string
  onCopy: (from: Profile, to: Profile, uniqueIds: string[]) => void
  onMatch: (from: Profile, to: Profile, rows: ComparePair[]) => void
}) {
  const { t } = useLingui()
  const [filter, setFilter] = useState('')
  const needle = filter.trim().toLowerCase()
  const { onlyA, onlyB, differentVersion, differentEnabled, identical, hasDiff } = filteredCompare(
    diff,
    needle,
  )
  const enabled = t`Enabled`
  const disabled = t`Off`
  const shared: SectionShared = { enabled, disabled, pending, lockedReason }

  if (!(needle || hasDiff) && identical.length > 0) {
    return (
      <>
        <CompareModFilter value={filter} onChange={setFilter} />
        <Typography
          sx={{ color: 'text.secondary' }}
        >{t`These profiles have the same mods.`}</Typography>
      </>
    )
  }

  return (
    <>
      <CompareModFilter value={filter} onChange={setFilter} />
      {onlyA.length > 0 ? (
        <OnlyInSection
          title={t`Only in ${aName}`}
          rows={onlyA}
          to={profileB}
          shared={shared}
          onCopy={(to, ids) => onCopy(profileA, to, ids)}
        />
      ) : null}
      {onlyB.length > 0 ? (
        <OnlyInSection
          title={t`Only in ${bName}`}
          rows={onlyB}
          to={profileA}
          shared={shared}
          onCopy={(to, ids) => onCopy(profileB, to, ids)}
        />
      ) : null}
      {differentVersion.length > 0 ? (
        <VersionSection
          rows={differentVersion}
          profileA={profileA}
          profileB={profileB}
          shared={shared}
          onMatch={onMatch}
        />
      ) : null}
      {differentEnabled.length > 0 ? (
        <CompareSection title={t`Different enabled state`}>
          {differentEnabled.map((row) => (
            <CompareDiffRow
              key={row.uniqueId}
              label={t`${row.name}: ${sideLabel(row.a, enabled, disabled)} → ${sideLabel(row.b, enabled, disabled)}`}
              aName={aName}
              bName={bName}
              pending={pending}
              copyToB={() => onCopy(profileA, profileB, [row.uniqueId])}
              copyToA={() => onCopy(profileB, profileA, [row.uniqueId])}
            />
          ))}
        </CompareSection>
      ) : null}
      <IdenticalList rows={identical} />
      {needle !== '' && !hasDiff && identical.length === 0 ? (
        <Typography sx={{ color: 'text.secondary' }}>{t`No matching mods`}</Typography>
      ) : null}
      {!hasDiff && identical.length === 0 && !needle ? (
        <EmptyState
          compact={true}
          icon={<Inbox size={28} />}
          title={t`No user mods in either profile.`}
        >
          {t`These profiles have no user mods to compare.`}
        </EmptyState>
      ) : null}
    </>
  )
}
