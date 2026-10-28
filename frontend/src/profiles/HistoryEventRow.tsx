import { useLingui } from '@lingui/react/macro'
import { Box, Button, Checkbox, Chip, Typography } from '@mui/material'
import {
  ArrowUpCircle,
  CircleDot,
  MinusCircle,
  PlusCircle,
  Settings2,
  ShieldCheck,
  ToggleLeft,
  ToggleRight,
} from 'lucide-react'
import { memo, type ReactNode, useState } from 'react'
import type {
  HistoryEvent,
  HistoryItem,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { When } from '../i18n/When.tsx'
import { Tooltip } from '../shell/Tooltip.tsx'
import { space } from '../theme/density.ts'
import { itemLabel, itemModKey } from './historyDiff.ts'
import { type HistoryKind, historyKind, historySummary } from './historyTimeline.ts'
import { ShowModsToggle } from './ShowModsToggle.tsx'

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

// Everything in a row but its compare checkbox, memoised so entering compare mode leaves it alone.
const RowMain = memo(function RowMainView({
  ev,
  items,
  busy,
  undoable,
  onUndo,
  onRevertItem,
}: {
  ev: HistoryEvent
  items: HistoryItem[]
  busy: boolean
  // False for the oldest event, which nothing older can undo.
  undoable: boolean
  onUndo: (id: string) => void
  onRevertItem: (id: string, mod: string) => void
}) {
  const { t } = useLingui()
  const [open, setOpen] = useState(false)
  const summary = historySummary(ev)
  const trimmed = ev.kind === 'trimmed'
  const detailed = items.length > 0
  return (
    <Box sx={{ flex: 1, minWidth: 0 }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap, minHeight: 32 }}>
        <Box sx={{ display: 'flex', color: 'text.secondary' }}>{ICONS[historyKind(ev)]}</Box>
        <Box sx={{ flex: 1, minWidth: 0 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap, flexWrap: 'wrap' }}>
            <Typography sx={{ fontSize: 'medium.fontSize', fontWeight: 600 }}>{summary}</Typography>
            {ev.kind === 'good' ? (
              <Chip size="small" color="success" variant="outlined" label={t`Known good`} />
            ) : null}
          </Box>
          <Typography sx={{ fontSize: 'small.fontSize', color: 'text.secondary' }}>
            <When value={ev.at} withTime={true} />
          </Typography>
        </Box>
        {trimmed || !undoable ? (
          // The same button, hidden, so every row's columns line up whether or not it can be undone.
          <Button
            size="small"
            disabled={true}
            aria-hidden={true}
            tabIndex={-1}
            sx={{ visibility: 'hidden' }}
          >
            {t`Undo these changes`}
          </Button>
        ) : (
          <Tooltip
            describeChild={true}
            title={t`Put the profile back as it was before this entry. Newer entries are undone too.`}
          >
            <span>
              <Button
                size="small"
                className="history-undo"
                disabled={busy}
                onClick={() => onUndo(ev.id)}
                aria-label={t`Undo these changes: ${summary}`}
                sx={{ opacity: 0, '&:focus-visible': { opacity: 1 } }}
              >
                {t`Undo these changes`}
              </Button>
            </span>
          </Tooltip>
        )}
        {detailed ? <ShowModsToggle open={open} onToggle={() => setOpen(!open)} /> : null}
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
              <Typography sx={{ fontSize: 'base.fontSize', flex: 1, minWidth: 0 }}>
                {itemLabel(item)}
              </Typography>
              <Tooltip
                describeChild={true}
                title={t`Undo only this mod's change from this entry. Everything else stays as it is.`}
              >
                <span>
                  <Button
                    size="small"
                    disabled={busy}
                    onClick={() => onRevertItem(ev.id, itemModKey(item))}
                    aria-label={t`Revert this mod: ${{ detail: itemLabel(item) }}`}
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
})

export const HistoryEventRow = memo(function HistoryEventRowView({
  ev,
  items,
  comparing,
  selected,
  busy,
  undoable,
  onToggle,
  onUndo,
  onRevertItem,
}: {
  ev: HistoryEvent
  items: HistoryItem[]
  // Compare mode shows a checkbox on each row that can be picked.
  comparing: boolean
  selected: boolean
  busy: boolean
  undoable: boolean
  onToggle: (id: string) => void
  onUndo: (id: string) => void
  onRevertItem: (id: string, mod: string) => void
}) {
  const { t } = useLingui()
  return (
    <Box
      component="li"
      sx={{
        listStyle: 'none',
        borderRadius: '8px',
        px: space.gap,
        py: 0.75,
        display: 'flex',
        alignItems: 'flex-start',
        gap: space.gap,
        [REVEAL]: { bgcolor: 'var(--mortar-card-hover)' },
        [`${REVEAL} .history-undo`]: { opacity: 1 },
      }}
    >
      {comparing ? (
        <Checkbox
          size="small"
          checked={selected}
          onChange={() => onToggle(ev.id)}
          disabled={busy || ev.kind === 'trimmed'}
          slotProps={{ input: { 'aria-label': t`Compare ${{ summary: historySummary(ev) }}` } }}
          sx={{ p: 0.5, mt: 0.25 }}
        />
      ) : null}
      <RowMain
        ev={ev}
        items={items}
        busy={busy}
        undoable={undoable}
        onUndo={onUndo}
        onRevertItem={onRevertItem}
      />
    </Box>
  )
})
