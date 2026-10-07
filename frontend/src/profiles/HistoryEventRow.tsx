import { useLingui } from '@lingui/react/macro'
import { Box, Button, Checkbox, Chip, Tooltip, Typography } from '@mui/material'
import {
  ArrowUpCircle,
  ChevronDown,
  ChevronRight,
  CircleDot,
  MinusCircle,
  PlusCircle,
  Settings2,
  ShieldCheck,
  ToggleLeft,
  ToggleRight,
} from 'lucide-react'
import { type ReactNode, useState } from 'react'
import type {
  HistoryEvent,
  HistoryItem,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { When } from '../i18n/When.tsx'
import { space } from '../theme/density.ts'
import { itemModKey } from './historyDiff.ts'
import { type HistoryKind, historyKind, historySummary, sentence } from './historyTimeline.ts'

const ICON_SIZE = 18
const REVEAL = '&:hover, &:focus-within'

const ICONS: Record<HistoryKind, ReactNode> = {
  added: <PlusCircle size={ICON_SIZE} />,
  removed: <MinusCircle size={ICON_SIZE} />,
  updated: <ArrowUpCircle size={ICON_SIZE} />,
  enabled: <ToggleRight size={ICON_SIZE} />,
  disabled: <ToggleLeft size={ICON_SIZE} />,
  settings: <Settings2 size={ICON_SIZE} />,
  knownGood: <ShieldCheck size={ICON_SIZE} />,
  other: <CircleDot size={ICON_SIZE} />,
}

export function HistoryEventRow({
  ev,
  items,
  comparing,
  selected,
  onToggle,
  onUndo,
  onRevertItem,
  busy,
}: {
  ev: HistoryEvent
  items: HistoryItem[]
  // Compare mode shows a checkbox on each row that can be picked.
  comparing: boolean
  selected: boolean
  onToggle: () => void
  // Absent for the oldest event, which nothing older can undo.
  onUndo: (() => void) | undefined
  onRevertItem: (mod: string) => void
  busy: boolean
}) {
  const { t } = useLingui()
  const [open, setOpen] = useState(false)
  const summary = historySummary(ev)
  const trimmed = ev.kind === 'trimmed'
  const detailed = items.length > 0
  return (
    <Box
      component="li"
      sx={{
        listStyle: 'none',
        borderRadius: '8px',
        px: space.gap,
        py: 0.75,
        [REVEAL]: { bgcolor: 'var(--mortar-card-hover)' },
        [`${REVEAL} .history-undo`]: { opacity: 1 },
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap, minHeight: 32 }}>
        {comparing ? (
          <Checkbox
            size="small"
            checked={selected}
            onChange={onToggle}
            disabled={busy || trimmed}
            slotProps={{ input: { 'aria-label': t`Compare ${summary}` } }}
            sx={{ p: 0.5 }}
          />
        ) : null}
        <Box sx={{ display: 'flex', color: 'text.secondary' }}>{ICONS[historyKind(ev)]}</Box>
        <Box sx={{ flex: 1, minWidth: 0 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap, flexWrap: 'wrap' }}>
            <Typography sx={{ fontSize: 14, fontWeight: 600 }}>{summary}</Typography>
            {ev.kind === 'good' ? (
              <Chip size="small" color="success" variant="outlined" label={t`Known good`} />
            ) : null}
          </Box>
          <Typography sx={{ fontSize: 12, color: 'text.secondary' }}>
            <When value={ev.at} withTime={true} />
          </Typography>
        </Box>
        {detailed ? (
          <Button
            size="small"
            aria-expanded={open}
            onClick={() => setOpen(!open)}
            startIcon={open ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
          >
            {open ? t`Hide mods` : t`Show mods`}
          </Button>
        ) : null}
        {trimmed || onUndo === undefined ? null : (
          <Tooltip
            describeChild={true}
            title={t`Put the profile back as it was before this entry. Newer entries are undone too.`}
          >
            <span>
              <Button
                size="small"
                className="history-undo"
                disabled={busy}
                onClick={onUndo}
                aria-label={t`Undo these changes: ${summary}`}
                sx={{ opacity: 0, '&:focus-visible': { opacity: 1 } }}
              >
                {t`Undo these changes`}
              </Button>
            </span>
          </Tooltip>
        )}
      </Box>
      {open ? (
        <Box component="ul" sx={{ m: 0, mt: 0.5, p: 0, pl: 5.5 }}>
          {items.map((item) => (
            <Box
              component="li"
              key={`${item.kind}:${item.mod}:${item.file ?? ''}`}
              sx={{
                listStyle: 'none',
                display: 'flex',
                alignItems: 'center',
                gap: space.gap,
                py: 0.25,
              }}
            >
              <Typography sx={{ fontSize: 13, flex: 1, minWidth: 0 }}>
                {sentence(item.detail)}
              </Typography>
              <Tooltip
                describeChild={true}
                title={t`Undo only this mod's change from this entry. Everything else stays as it is.`}
              >
                <span>
                  <Button
                    size="small"
                    disabled={busy}
                    onClick={() => onRevertItem(itemModKey(item))}
                    aria-label={t`Revert this mod: ${sentence(item.detail)}`}
                  >
                    {t`Revert this mod`}
                  </Button>
                </span>
              </Tooltip>
            </Box>
          ))}
        </Box>
      ) : null}
    </Box>
  )
}
