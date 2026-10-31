import { useLingui } from '@lingui/react/macro'
import { Box, ToggleButton } from '@mui/material'
import {
  Code,
  Folder,
  Frame,
  Image,
  Lightbulb,
  type LucideIcon,
  Music,
  Shirt,
  Tv,
  Volume2,
} from 'lucide-react'
import type { Param } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import { Field } from '../shell/dialog/FormDialog.tsx'
import { space } from '../theme/density.ts'
import type { CodeTile } from './Params.tsx'

// A folder the catalog adds later still gets a tile, with the plain folder icon.
const ICONS: Record<string, LucideIcon> = {
  posters: Image,
  tips: Lightbulb,
  paintings: Frame,
  boombox: Music,
  tv: Tv,
  suits: Shirt,
  sounds: Volume2,
}

function Tile({
  icon: Icon,
  label,
  on,
  toggle,
}: {
  icon: LucideIcon
  label: string
  on: boolean
  toggle: () => void
}) {
  return (
    <ToggleButton
      value={label}
      selected={on}
      onChange={toggle}
      sx={{ justifyContent: 'flex-start', gap: space.gap, textTransform: 'none' }}
    >
      <Icon size={18} aria-hidden={true} style={{ flexShrink: 0 }} />
      {label}
    </ToggleButton>
  )
}

// What a project carries, as a grid of toggles: its code, when it may go without, then its content folders.
export function ContentTicks({
  param,
  value,
  set,
  code,
}: {
  param: Param
  value: string
  set: (v: string) => void
  code?: CodeTile | undefined
}) {
  const { t } = useLingui()
  const choices = param.choices ?? []
  const picked = value.split('|').filter(Boolean)
  const tick = (id: string, on: boolean) =>
    set(
      choices
        .map((c) => c.value)
        .filter((v) => (v === id ? on : picked.includes(v)))
        .join('|'),
    )
  return (
    <Field label={param.label} hint={param.description}>
      <Box
        role="group"
        aria-label={param.label}
        sx={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fill, minmax(140px, 1fr))',
          gap: space.gap,
        }}
      >
        {code ? (
          <Tile icon={Code} label={t`Code`} on={code.on} toggle={() => code.set(!code.on)} />
        ) : null}
        {choices.map((c) => (
          <Tile
            key={c.value}
            icon={ICONS[c.value] ?? Folder}
            label={c.label}
            on={picked.includes(c.value)}
            toggle={() => tick(c.value, !picked.includes(c.value))}
          />
        ))}
      </Box>
    </Field>
  )
}
