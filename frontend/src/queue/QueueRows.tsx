import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
import { ArrowUpToLine, ChevronDown, ChevronUp, Pause, Play, X } from 'lucide-react'
import type { ReactNode } from 'react'
import type { Item } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/queue/models.ts'
import {
  Move,
  PauseItem,
  ResumeItem,
  Skip,
  Top,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/queue/service.ts'
import { formatBytes, formatKb } from '../i18n/bytes.ts'
import { LetterTile } from '../mods/parts.tsx'
import { TipIconButton } from '../shell/TipIconButton.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { Title } from './Callout.tsx'
import { detail, ROW } from './rowStyle.ts'
import { tile } from './totals.ts'

export function SectionTitle({
  color,
  children,
  action,
}: {
  color?: string
  children: ReactNode
  action?: ReactNode
}) {
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        fontSize: 12,
        fontWeight: 700,
        letterSpacing: '0.06em',
        textTransform: 'uppercase',
        color: color ?? 'var(--mortar-ink-sec)',
      }}
    >
      {children}
      {action}
    </Box>
  )
}

export function Row({
  item,
  sub,
  actions,
  sx,
}: {
  item: Item
  sub: ReactNode
  actions: ReactNode
  sx?: object
}) {
  return (
    <Box sx={{ ...ROW, ...sx }}>
      <LetterTile mod={tile(item)} size={36} />
      <Box sx={{ flexGrow: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }}>
        <Title item={item} size={14} />
        {sub}
      </Box>
      {actions}
    </Box>
  )
}

export function PauseButton({ item }: { item: Item }) {
  const { t } = useLingui()
  return (
    <TipIconButton
      label={t`Pause ${item.name}`}
      onClick={() => PauseItem(item.id).catch(reportUnexpected)}
    >
      <Pause size={14} />
    </TipIconButton>
  )
}

export function NowButton({ item }: { item: Item }) {
  const { t } = useLingui()
  return (
    <TipIconButton
      label={t`Download ${item.name} now`}
      onClick={() => Top(item.id).catch(reportUnexpected)}
    >
      <ArrowUpToLine size={14} />
    </TipIconButton>
  )
}

export function MoveButtons({ item }: { item: Item }) {
  const { t } = useLingui()
  return (
    <>
      <TipIconButton
        label={t`Move ${item.name} up`}
        onClick={() => Move(item.id, -1).catch(reportUnexpected)}
      >
        <ChevronUp size={14} />
      </TipIconButton>
      <TipIconButton
        label={t`Move ${item.name} down`}
        onClick={() => Move(item.id, 1).catch(reportUnexpected)}
      >
        <ChevronDown size={14} />
      </TipIconButton>
    </>
  )
}

// A paused download: how much is already on disk, or that it waits for a slot after making way for another.
export function Paused({ items }: { items: Item[] }) {
  const { t } = useLingui()
  if (items.length === 0) {
    return null
  }
  return (
    <>
      <SectionTitle>{t`Paused (${items.length})`}</SectionTitle>
      {items.map((i) => {
        const done = formatBytes(i.partial ?? 0)
        const total = formatKb(i.sizeKb)
        return (
          <Row
            key={i.id}
            item={i}
            sx={{ bgcolor: 'var(--mortar-raised-60)' }}
            sub={
              <Typography sx={{ ...detail, color: 'text.secondary' }}>
                {i.autoResume
                  ? t`Waiting for a free slot · ${done} of ${total}`
                  : t`Paused at ${done} of ${total}`}
              </Typography>
            }
            actions={
              <>
                <TipIconButton
                  label={t`Resume ${i.name}`}
                  onClick={() => ResumeItem(i.id).catch(reportUnexpected)}
                >
                  <Play size={14} />
                </TipIconButton>
                <NowButton item={i} />
                <TipIconButton
                  label={t`Skip ${i.name}`}
                  onClick={() => Skip(i.id).catch(reportUnexpected)}
                >
                  <X size={14} />
                </TipIconButton>
              </>
            }
          />
        )
      })}
    </>
  )
}
