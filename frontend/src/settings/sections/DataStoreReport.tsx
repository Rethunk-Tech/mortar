import type { I18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Checkbox, FormControlLabel } from '@mui/material'
import { useEffect, useState } from 'react'
import {
  RemoveItems,
  Report,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/datasvc/service.ts'
import type {
  Item,
  Report as StoreReport,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/store/models.ts'
import { formatBytes } from '../../i18n/bytes.ts'
import { When } from '../../i18n/When.tsx'
import { ConfirmDialog } from '../../shell/ConfirmDialog.tsx'
import { reportUnexpected } from '../../toasts/report.ts'
import { nowrap } from './dataStyles.ts'

const SELECT_SEP = '\u0000'
const TITLE_SIZE = 14
const HINT_SIZE = 13

interface Sel {
  game: string
  item: Item
}
interface DupGroup {
  game: string
  items: Item[]
}

function selId(game: string, key: string) {
  return `${game}${SELECT_SEP}${key}`
}

function newestOf(group: Item[]): Item {
  return group.reduce((best, cur) =>
    Date.parse(String(cur.lastUsed)) > Date.parse(String(best.lastUsed)) ? cur : best,
  )
}

function confirmBody(i18n: I18n, count: number, size: string) {
  return i18n._(msg`Remove ${count} selected items? This frees ${size}.`)
}

function flattenReport(rep: StoreReport) {
  const unused: Sel[] = []
  const dups: DupGroup[] = []
  for (const [game, g] of Object.entries(rep ?? {})) {
    if (g) {
      for (const item of g.unused ?? []) {
        unused.push({ game, item })
      }
      for (const items of g.duplicates ?? []) {
        if (items && items.length > 1) {
          dups.push({ game, items })
        }
      }
    }
  }
  return { unused, dups }
}

function selectedFrom(picked: string[], unused: Sel[], dups: DupGroup[]): Sel[] {
  const byId = new Map<string, Sel>()
  for (const row of unused) {
    byId.set(selId(row.game, row.item.key), row)
  }
  for (const g of dups) {
    for (const item of g.items) {
      byId.set(selId(g.game, item.key), { game: g.game, item })
    }
  }
  return picked.flatMap((id) => {
    const hit = byId.get(id)
    return hit ? [hit] : []
  })
}

function UnusedList({
  rows,
  picked,
  onToggle,
}: {
  rows: Sel[]
  picked: string[]
  onToggle: (id: string, on: boolean) => void
}) {
  const { t } = useLingui()
  if (rows.length === 0) {
    return null
  }
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
      <Box sx={{ fontSize: TITLE_SIZE }}>{t`Not used by any profile`}</Box>
      {rows.map((row) => {
        const id = selId(row.game, row.item.key)
        return (
          <ItemRow
            key={id}
            item={row.item}
            checked={picked.includes(id)}
            onToggle={(on) => onToggle(id, on)}
          />
        )
      })}
    </Box>
  )
}

function DupList({
  groups,
  picked,
  onToggle,
}: {
  groups: DupGroup[]
  picked: string[]
  onToggle: (id: string, on: boolean) => void
}) {
  const { t } = useLingui()
  if (groups.length === 0) {
    return null
  }
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
      <Box sx={{ fontSize: TITLE_SIZE }}>{t`Same mod stored twice`}</Box>
      {groups.map((g) => {
        const keep = newestOf(g.items)
        return (
          <Box key={`${g.game}-${g.items.map((i) => i.key).join('-')}`}>
            <Box sx={{ fontSize: HINT_SIZE, color: 'text.secondary' }}>
              {t`Keep the newest copy (${keep.name} ${keep.version})`}
            </Box>
            {g.items.map((item) => {
              const id = selId(g.game, item.key)
              return (
                <ItemRow
                  key={id}
                  item={item}
                  checked={picked.includes(id)}
                  onToggle={(on) => onToggle(id, on)}
                />
              )
            })}
          </Box>
        )
      })}
    </Box>
  )
}

function ItemRow({
  item,
  checked,
  onToggle,
}: {
  item: Item
  checked: boolean
  onToggle: (on: boolean) => void
}) {
  const { name, key, version, size, lastUsed } = item
  return (
    <FormControlLabel
      sx={{ ...nowrap, alignItems: 'center' }}
      control={
        <Checkbox
          checked={checked}
          onChange={(ev) => {
            const { checked: on } = ev.target
            onToggle(on)
          }}
        />
      }
      label={
        <Box sx={{ display: 'flex', gap: 1, alignItems: 'center', fontSize: TITLE_SIZE }}>
          <Box>{name || key}</Box>
          <Box sx={{ color: 'text.secondary' }}>{version}</Box>
          <Box sx={{ color: 'text.secondary' }}>{formatBytes(size)}</Box>
          {lastUsed ? <When value={lastUsed} /> : null}
        </Box>
      }
    />
  )
}

function DataStoreReport({ onChanged }: { onChanged: () => void }) {
  const { t, i18n } = useLingui()
  const [unused, setUnused] = useState<Sel[]>([])
  const [dups, setDups] = useState<DupGroup[]>([])
  const [picked, setPicked] = useState<string[]>([])
  const [confirm, setConfirm] = useState(false)
  const [busy, setBusy] = useState(false)
  useEffect(() => {
    Report()
      .then((rep) => {
        const next = flattenReport(rep)
        setUnused(next.unused)
        setDups(next.dups)
      })
      .catch(reportUnexpected)
  }, [])
  const chosen = selectedFrom(picked, unused, dups)
  const bytes = chosen.reduce((n, row) => n + row.item.size, 0)
  const toggle = (id: string, on: boolean) => {
    setPicked((cur) => (on ? [...cur, id] : cur.filter((x) => x !== id)))
  }
  const run = () => {
    const byGame = new Map<string, string[]>()
    for (const row of chosen) {
      byGame.set(row.game, [...(byGame.get(row.game) ?? []), row.item.key])
    }
    setBusy(true)
    Promise.all([...byGame.entries()].map(([game, keys]) => RemoveItems(game, keys)))
      .then(() => {
        setConfirm(false)
        setPicked([])
        onChanged()
        return Report()
      })
      .then((rep) => {
        const next = flattenReport(rep)
        setUnused(next.unused)
        setDups(next.dups)
      })
      .catch(reportUnexpected)
      .finally(() => setBusy(false))
  }
  const empty = unused.length === 0 && dups.length === 0
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      <Box sx={{ fontSize: TITLE_SIZE, fontWeight: 600, pt: 1 }}>{t`Store cleanup`}</Box>
      {empty ? (
        <Box sx={{ fontSize: TITLE_SIZE }}>{t`No unused or duplicate store items.`}</Box>
      ) : null}
      <UnusedList rows={unused} picked={picked} onToggle={toggle} />
      <DupList groups={dups} picked={picked} onToggle={toggle} />
      <Button
        disabled={chosen.length === 0}
        onClick={() => setConfirm(true)}
        sx={{ alignSelf: 'flex-start', ...nowrap }}
      >
        {t`Remove selected`}
      </Button>
      <ConfirmDialog
        open={confirm}
        title={t`Remove selected store items?`}
        body={confirmBody(i18n, chosen.length, formatBytes(bytes))}
        confirmLabel={t`Remove`}
        color="error"
        busy={busy}
        onCancel={() => setConfirm(false)}
        onConfirm={run}
      />
    </Box>
  )
}

export { DataStoreReport }
