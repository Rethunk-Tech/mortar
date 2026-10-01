import { useLingui } from '@lingui/react/macro'
import { Box, Collapse, IconButton, Typography } from '@mui/material'
import { ChevronDown, ChevronRight } from 'lucide-react'
import { useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { CompareDiffRow, CompareSection } from './CompareRows.tsx'
import type { ComparePair, CompareSide, ProfileCompare } from './compare.ts'
import { sideLabel } from './compare.ts'

function SameModsMessage() {
  const { t } = useLingui()
  return (
    <Typography
      sx={{ color: 'text.secondary' }}
    >{t`These profiles have the same mods.`}</Typography>
  )
}

function EmptyModsMessage() {
  const { t } = useLingui()
  return (
    <Typography sx={{ color: 'text.secondary' }}>{t`No user mods in either profile.`}</Typography>
  )
}

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
          onClick={() => setOpen((v) => !v)}
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

export function CompareBody({
  diff,
  profileA,
  profileB,
  aName,
  bName,
  pending,
  onCopy,
}: {
  diff: ProfileCompare
  profileA: Profile
  profileB: Profile
  aName: string
  bName: string
  pending: boolean
  onCopy: (from: Profile, to: Profile, uniqueId: string) => void
}) {
  const { t } = useLingui()
  const enabled = t`Enabled`
  const disabled = t`Switched off`
  const hasDiff =
    diff.onlyA.length > 0 ||
    diff.onlyB.length > 0 ||
    diff.differentVersion.length > 0 ||
    diff.differentEnabled.length > 0

  const pairRow = (row: ComparePair) => (
    <CompareDiffRow
      key={row.uniqueId}
      label={t`${row.name}: ${sideLabel(row.a, enabled, disabled)} → ${sideLabel(row.b, enabled, disabled)}`}
      aName={aName}
      bName={bName}
      pending={pending}
      copyToB={() => onCopy(profileA, profileB, row.uniqueId)}
      copyToA={() => onCopy(profileB, profileA, row.uniqueId)}
    />
  )

  const onlyRow = (side: CompareSide, from: Profile, to: Profile, direction: 'toB' | 'toA') => (
    <CompareDiffRow
      key={side.uniqueId}
      label={sideLabel(side, enabled, disabled)}
      aName={aName}
      bName={bName}
      pending={pending}
      copyToB={direction === 'toB' ? () => onCopy(from, to, side.uniqueId) : undefined}
      copyToA={direction === 'toA' ? () => onCopy(from, to, side.uniqueId) : undefined}
    />
  )

  if (!hasDiff && diff.identical.length > 0) {
    return <SameModsMessage />
  }

  return (
    <>
      {diff.onlyA.length > 0 ? (
        <CompareSection title={t`Only in ${aName}`}>
          {diff.onlyA.map((side) => onlyRow(side, profileA, profileB, 'toB'))}
        </CompareSection>
      ) : null}
      {diff.onlyB.length > 0 ? (
        <CompareSection title={t`Only in ${bName}`}>
          {diff.onlyB.map((side) => onlyRow(side, profileB, profileA, 'toA'))}
        </CompareSection>
      ) : null}
      {diff.differentVersion.length > 0 ? (
        <CompareSection title={t`Different version`}>
          {diff.differentVersion.map((row) => pairRow(row))}
        </CompareSection>
      ) : null}
      {diff.differentEnabled.length > 0 ? (
        <CompareSection title={t`Different enabled state`}>
          {diff.differentEnabled.map((row) => pairRow(row))}
        </CompareSection>
      ) : null}
      <IdenticalList rows={diff.identical} />
      {!hasDiff && diff.identical.length === 0 ? <EmptyModsMessage /> : null}
    </>
  )
}
