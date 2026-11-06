import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Chip, Link, Typography } from '@mui/material'
import { TriangleAlert } from 'lucide-react'
import { useState } from 'react'
import type { Row } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/loadorder/models.ts'
import { useTab } from '../game/tab.ts'
import { idKey, localId } from './dependents.ts'
import { sameId } from './lookup.ts'
import { focusMod } from './openMod.ts'
import { requirementNameIn } from './requirementName.ts'
import { useMods } from './store.ts'

const INLINE_REQUIRED = 4
const INLINE_USERS = 2

function DepChip({
  label,
  title,
  missing = false,
  expanded,
  onClick,
}: {
  label: string
  title?: string
  missing?: boolean
  expanded?: boolean
  onClick?: () => void
}) {
  return (
    <Chip
      size="small"
      variant="outlined"
      label={label}
      title={title}
      color={missing ? 'error' : 'default'}
      icon={missing ? <TriangleAlert size={12} aria-hidden={true} /> : undefined}
      aria-expanded={expanded}
      onClick={onClick}
      sx={{ maxWidth: 240 }}
    />
  )
}

// Whether the row's mod is in the profile, not the mod: the store rebuilds every mod on each change, and a row shows
// none of it.
export function RowName({ row }: { row: Row }) {
  const { t } = useLingui()
  const listed = useMods((s) => s.mods.some((m) => sameId(m.id, row.id)))
  const known = row.name.trim() !== ''
  const label = known ? row.name : localId(row.id) || t`Unknown mod`
  const title = label
  if (!listed) {
    return (
      <Typography noWrap={true} title={title} sx={{ fontWeight: 600, minWidth: 0 }}>
        {label}
      </Typography>
    )
  }
  return (
    <Link
      component="button"
      noWrap={true}
      title={title}
      onClick={() => {
        const mod = useMods.getState().mods.find((m) => sameId(m.id, row.id))
        if (mod) {
          useTab.getState().setTab('mods')
          focusMod(mod)
        }
      }}
      sx={{ fontWeight: 600, minWidth: 0, color: 'text.primary', textAlign: 'left' }}
    >
      {label}
    </Link>
  )
}

// The profile's names for ids the order did not name, one per line: a string, so a mod change that leaves these names
// alone leaves the row alone too.
const profileNames = (s: Parameters<typeof requirementNameIn>[0], ids: string[]) =>
  ids.map((id) => requirementNameIn(s, id)).join('\n')

export function DepChips({
  row,
  names,
  onScroll,
}: {
  row: Row
  names: Map<string, string>
  onScroll: (id: string) => void
}) {
  const { t } = useLingui()
  const [open, setOpen] = useState(false)
  const unnamed = [
    ...(row.required ?? []),
    ...(row.optional ?? []),
    ...(row.dependents ?? []),
  ].filter((id) => (names.get(idKey(id)) ?? '').trim() === '')
  const looked = useMods((s) => profileNames(s, unnamed)).split('\n')
  const chip = (prefix: 'req' | 'opt' | 'dep', id: string, missing = false) => {
    const name = (names.get(idKey(id)) ?? '').trim()
    const known = name !== ''
    const label = known ? name : (looked[unnamed.indexOf(id)] ?? id)
    const text = {
      req: missing ? t`Needs ${{ name: label }} (missing)` : t`Required: ${{ name: label }}`,
      opt: t`Optional: ${{ name: label }}`,
      dep: t`Used by: ${{ name: label }}`,
    }[prefix]
    return (
      <DepChip
        key={`${prefix}-${id}`}
        label={text}
        missing={missing}
        {...(known ? {} : { title: id })}
        {...(known && !missing ? { onClick: () => onScroll(id) } : {})}
      />
    )
  }
  const missingIds = new Set((row.missingRequired ?? []).map(idKey))
  const required = row.required ?? []
  const optional = row.optional ?? []
  const users = row.dependents ?? []
  const long = required.length > INLINE_REQUIRED || users.length > INLINE_USERS
  const collapsedLabel = [
    users.length > INLINE_USERS
      ? plural({ count: users.length }, { one: 'Used by: # mod', other: 'Used by: # mods' })
      : '',
    required.length > INLINE_REQUIRED
      ? t`${{ count: required.length - INLINE_REQUIRED }} more required`
      : '',
  ]
    .filter((part) => part !== '')
    .join(' · ')
  const shownRequired = open ? required : required.slice(0, INLINE_REQUIRED)
  if (required.length + optional.length + users.length === 0) {
    return null
  }
  return (
    <>
      {shownRequired.map((id) => chip('req', id, missingIds.has(idKey(id))))}
      {optional.map((id) => chip('opt', id))}
      {open || users.length <= INLINE_USERS ? users.map((id) => chip('dep', id)) : null}
      {long ? (
        <DepChip
          label={open ? t`Show less` : collapsedLabel}
          expanded={open}
          onClick={() => setOpen((v) => !v)}
        />
      ) : null}
    </>
  )
}
