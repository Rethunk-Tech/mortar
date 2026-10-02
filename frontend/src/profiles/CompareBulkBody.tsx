import { useLingui } from '@lingui/react/macro'
import { Box, Button, Collapse, IconButton, Typography } from '@mui/material'
import { ChevronDown, ChevronRight } from 'lucide-react'
import type { ReactNode } from 'react'
import { useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
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
        {t`Use A's version in B`}
      </Button>
      <Button
        size="small"
        variant="outlined"
        disabled={pending}
        onClick={() => onCopy(profileB, profileA, row.uniqueId)}
        sx={{ whiteSpace: 'nowrap', flexShrink: 0 }}
      >
        {t`Use B's version in A`}
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
  const enabled = t`Enabled`
  const disabled = t`Switched off`
  const hasDiff =
    diff.onlyA.length > 0 ||
    diff.onlyB.length > 0 ||
    diff.differentVersion.length > 0 ||
    diff.differentEnabled.length > 0

  const onlySection = ({
    title,
    rows,
    from,
    to,
    direction,
  }: {
    title: string
    rows: CompareSide[]
    from: Profile
    to: Profile
    direction: 'toB' | 'toA'
  }) => (
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

  if (!hasDiff && diff.identical.length > 0) {
    return (
      <Typography
        sx={{ color: 'text.secondary' }}
      >{t`These profiles have the same mods.`}</Typography>
    )
  }

  return (
    <>
      {diff.onlyA.length > 0
        ? onlySection({
            title: t`Only in ${aName}`,
            rows: diff.onlyA,
            from: profileA,
            to: profileB,
            direction: 'toB',
          })
        : null}
      {diff.onlyB.length > 0
        ? onlySection({
            title: t`Only in ${bName}`,
            rows: diff.onlyB,
            from: profileB,
            to: profileA,
            direction: 'toA',
          })
        : null}
      {diff.differentVersion.length > 0 ? (
        <CompareSection title={t`Different version`}>
          {diff.differentVersion.map((row) => (
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
      {diff.differentEnabled.length > 0 ? (
        <CompareSection title={t`Different enabled state`}>
          {diff.differentEnabled.map((row) => (
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
      <IdenticalList rows={diff.identical} />
      {!hasDiff && diff.identical.length === 0 ? (
        <Typography
          sx={{ color: 'text.secondary' }}
        >{t`No user mods in either profile.`}</Typography>
      ) : null}
    </>
  )
}
