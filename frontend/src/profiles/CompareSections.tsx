import { useLingui } from '@lingui/react/macro'
import { Box, Button, Checkbox, FormControlLabel, Typography } from '@mui/material'
import type { ReactNode } from 'react'
import { useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import type { ComparePair, CompareSide, SectionShared } from './compare.ts'
import { sideLabel } from './compare.ts'

function SectionHeader({
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

function ActionButton({
  label,
  to,
  disabled,
  shared,
  onClick,
}: {
  label: string
  to: Profile
  disabled: boolean
  shared: SectionShared
  onClick: () => void
}) {
  const reason = shared.lockedReason(to)
  return (
    <DisabledReason title={reason} disabled={reason !== ''}>
      <Button
        size="small"
        variant="outlined"
        disabled={shared.pending || disabled || reason !== ''}
        onClick={onClick}
      >
        {label}
      </Button>
    </DisabledReason>
  )
}

function OnlyInSection({
  title,
  rows,
  to,
  shared,
  onCopy,
}: {
  title: string
  rows: CompareSide[]
  to: Profile
  shared: SectionShared
  onCopy: (to: Profile, ids: string[]) => void
}) {
  const { t } = useLingui()
  const [unchecked, setUnchecked] = useState<ReadonlySet<string>>(new Set())
  const picked = rows.filter((row) => !unchecked.has(row.id)).map((row) => row.id)
  const toggle = (id: string) =>
    setUnchecked((prev) => {
      const next = new Set(prev)
      if (!next.delete(id)) {
        next.add(id)
      }
      return next
    })
  return (
    <SectionHeader
      title={title}
      action={
        <ActionButton
          label={t`Copy ${picked.length} to ${to.name}`}
          to={to}
          disabled={picked.length === 0}
          shared={shared}
          onClick={() => onCopy(to, picked)}
        />
      }
    >
      {rows.map((side) => (
        <FormControlLabel
          key={side.id}
          sx={{ display: 'flex', mx: 0, minHeight: 36 }}
          control={
            <Checkbox
              size="small"
              checked={!unchecked.has(side.id)}
              disabled={shared.pending}
              onChange={() => toggle(side.id)}
            />
          }
          label={
            <Typography noWrap={true} sx={{ fontSize: 14 }}>
              {sideLabel(side, shared.enabled, shared.disabled)}
            </Typography>
          }
        />
      ))}
    </SectionHeader>
  )
}

function VersionSection({
  rows,
  profileA,
  profileB,
  shared,
  onMatch,
}: {
  rows: ComparePair[]
  profileA: Profile
  profileB: Profile
  shared: SectionShared
  onMatch: (from: Profile, to: Profile, rows: ComparePair[]) => void
}) {
  const { t } = useLingui()
  return (
    <SectionHeader
      title={t`Different version`}
      action={
        <Box sx={{ display: 'flex', gap: 1 }}>
          <ActionButton
            label={t`Match versions in ${profileB.name}`}
            to={profileB}
            disabled={false}
            shared={shared}
            onClick={() => onMatch(profileA, profileB, rows)}
          />
          <ActionButton
            label={t`Match versions in ${profileA.name}`}
            to={profileA}
            disabled={false}
            shared={shared}
            onClick={() => onMatch(profileB, profileA, rows)}
          />
        </Box>
      }
    >
      {rows.map((row) => (
        <Typography
          key={row.id}
          noWrap={true}
          sx={{ fontSize: 14, py: 0.5, minHeight: 36 }}
          title={row.name}
        >
          {t`${row.name}: ${sideLabel(row.a, shared.enabled, shared.disabled)} → ${sideLabel(row.b, shared.enabled, shared.disabled)}`}
        </Typography>
      ))}
    </SectionHeader>
  )
}

export { OnlyInSection, VersionSection }
