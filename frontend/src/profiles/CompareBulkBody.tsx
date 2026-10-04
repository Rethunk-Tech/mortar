import { useLingui } from '@lingui/react/macro'
import { Box, Button, Collapse, IconButton, Typography } from '@mui/material'
import { ChevronDown, ChevronRight, Inbox } from 'lucide-react'
import type { ReactNode } from 'react'
import { useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { EmptyState } from '../shell/EmptyState.tsx'
import { SearchField } from '../shell/SearchField.tsx'
import { CompareDiffRow, CompareSection } from './CompareRows.tsx'
import type { ComparePair, CompareSide, ProfileCompare } from './compare.ts'
import { sideLabel } from './compare.ts'

function BulkSection({
  title,
  action,
  children,
}: {
  title: string
  action: ReactNode
  children: ReactNode
}) {
  return (
    <Box sx={{ mb: 2 }}>
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 1 }}>
        <Typography sx={{ fontSize: 13, fontWeight: 700, color: 'text.secondary' }}>
          {title}
        </Typography>
        {action}
      </Box>
      {children}
    </Box>
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

function OnlyInSection({
  title,
  rows,
  from,
  to,
  direction,
  aName,
  bName,
  enabled,
  disabled,
  pending,
  onCopy,
  onCopyAll,
}: {
  title: string
  rows: CompareSide[]
  from: Profile
  to: Profile
  direction: 'toB' | 'toA'
  aName: string
  bName: string
  enabled: string
  disabled: string
  pending: boolean
  onCopy: (from: Profile, to: Profile, uniqueId: string) => void
  onCopyAll: (from: Profile, to: Profile, uniqueIds: string[]) => void
}) {
  const { t } = useLingui()
  return (
    <BulkSection
      title={title}
      action={
        <Button
          size="small"
          variant="outlined"
          disabled={pending}
          onClick={() =>
            onCopyAll(
              from,
              to,
              rows.map((row) => row.uniqueId),
            )
          }
        >
          {t`Copy all to ${to.name}`}
        </Button>
      }
    >
      {rows.map((side) => (
        <CompareDiffRow
          key={side.uniqueId}
          label={sideLabel(side, enabled, disabled)}
          aName={aName}
          bName={bName}
          pending={pending}
          {...(direction === 'toB'
            ? { copyToB: () => onCopy(from, to, side.uniqueId) }
            : { copyToA: () => onCopy(from, to, side.uniqueId) })}
        />
      ))}
    </BulkSection>
  )
}

function VersionRow({
  row,
  enabled,
  disabled,
  pending,
  profileA,
  profileB,
  onCopy,
}: {
  row: ComparePair
  profileA: Profile
  profileB: Profile
  enabled: string
  disabled: string
  pending: boolean
  onCopy: (from: Profile, to: Profile, uniqueId: string) => void
}) {
  const { t } = useLingui()
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, py: 0.5, minHeight: 36 }}>
      <Typography sx={{ flex: 1, fontSize: 14, minWidth: 0 }} noWrap={true} title={row.name}>
        {t`${row.name}: ${sideLabel(row.a, enabled, disabled)} → ${sideLabel(row.b, enabled, disabled)}`}
      </Typography>
      <Button
        size="small"
        variant="outlined"
        disabled={pending}
        onClick={() => onCopy(profileA, profileB, row.uniqueId)}
        sx={{ whiteSpace: 'nowrap', flexShrink: 0 }}
      >
        {t`Use ${profileA.name}'s version in ${profileB.name}`}
      </Button>
      <Button
        size="small"
        variant="outlined"
        disabled={pending}
        onClick={() => onCopy(profileB, profileA, row.uniqueId)}
        sx={{ whiteSpace: 'nowrap', flexShrink: 0 }}
      >
        {t`Use ${profileB.name}'s version in ${profileA.name}`}
      </Button>
    </Box>
  )
}

export function CompareBulkBody({
  diff,
  profileA,
  profileB,
  aName,
  bName,
  pending,
  onCopy,
  onCopyAll,
}: {
  diff: ProfileCompare
  profileA: Profile
  profileB: Profile
  aName: string
  bName: string
  pending: boolean
  onCopy: (from: Profile, to: Profile, uniqueId: string) => void
  onCopyAll: (from: Profile, to: Profile, uniqueIds: string[]) => void
}) {
  const { t } = useLingui()
  const [filter, setFilter] = useState('')
  const needle = filter.trim().toLowerCase()
  const { onlyA, onlyB, differentVersion, differentEnabled, identical, hasDiff } = filteredCompare(
    diff,
    needle,
  )
  const enabled = t`Enabled`
  const disabled = t`Switched off`
  const sectionProps = {
    aName,
    bName,
    enabled,
    disabled,
    pending,
    onCopy,
    onCopyAll,
  }

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
          from={profileA}
          to={profileB}
          direction="toB"
          {...sectionProps}
        />
      ) : null}
      {onlyB.length > 0 ? (
        <OnlyInSection
          title={t`Only in ${bName}`}
          rows={onlyB}
          from={profileB}
          to={profileA}
          direction="toA"
          {...sectionProps}
        />
      ) : null}
      {differentVersion.length > 0 ? (
        <CompareSection title={t`Different version`}>
          {differentVersion.map((row) => (
            <VersionRow
              key={row.uniqueId}
              row={row}
              profileA={profileA}
              profileB={profileB}
              enabled={enabled}
              disabled={disabled}
              pending={pending}
              onCopy={onCopy}
            />
          ))}
        </CompareSection>
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
              copyToB={() => onCopy(profileA, profileB, row.uniqueId)}
              copyToA={() => onCopy(profileB, profileA, row.uniqueId)}
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
