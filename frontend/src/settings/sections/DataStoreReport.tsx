import type { I18n } from '@lingui/core'
import { msg, plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Checkbox,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Skeleton,
} from '@mui/material'
import { type ReactNode, useCallback, useEffect, useState } from 'react'
import type {
  Item as LeftoverItem,
  Preview,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/datasvc/models.ts'
import {
  Cleanup,
  CleanupPreview,
  RemoveItems,
  Report,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/datasvc/service.ts'
import type {
  Item,
  Report as StoreReport,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/store/models.ts'
import { formatBytes } from '../../i18n/bytes.ts'
import { When } from '../../i18n/When.tsx'
import { ConfirmDialog } from '../../shell/ConfirmDialog.tsx'
import { useToasts } from '../../toasts/store.ts'
import { usePending } from '../../toasts/usePending.ts'
import { nowrap } from './dataStyles.ts'

const SELECT_SEP = '\u0000'
const NEXUS_KEY = /^nexus-(\d+)-\d+$/

interface Sel {
  id: string
  game: string
  item: Item
}

type LeftoverKind = 'nexus' | 'github' | 'cache' | 'temp'

interface LeftoverGroup {
  kind: LeftoverKind
  items: LeftoverItem[]
  size: number
}

function newestOf(group: Item[]): Item {
  return group.reduce((best, cur) =>
    Date.parse(String(cur.lastUsed)) > Date.parse(String(best.lastUsed)) ? cur : best,
  )
}

// Removable store items: everything no profile uses, plus every copy of a duplicate except those a profile uses (the newest when none is).
function removable(rep: StoreReport): { unused: Sel[]; older: Sel[] } {
  const unused: Sel[] = []
  const older: Sel[] = []
  for (const [game, g] of Object.entries(rep ?? {})) {
    for (const item of g?.unused ?? []) {
      unused.push({ id: `${game}${SELECT_SEP}${item.key}`, game, item })
    }
    for (const items of (g?.duplicates ?? []).filter(
      (d): d is Item[] => d !== null && d.length > 1,
    )) {
      const used = items.filter((i) => i.inUse)
      const keep = used.length > 0 ? used : [newestOf(items)]
      for (const item of items.filter((i) => !keep.includes(i))) {
        older.push({ id: `${game}${SELECT_SEP}${item.key}`, game, item })
      }
    }
  }
  return { unused, older }
}

function leftoverKind(it: LeftoverItem): LeftoverKind {
  if (it.kind === 'temp') {
    return 'temp'
  }
  if (it.rel.startsWith('cache/nexus/')) {
    return 'nexus'
  }
  return it.rel.startsWith('cache/github-') ? 'github' : 'cache'
}

function groupLeftovers(preview: Preview | null): LeftoverGroup[] {
  const groups = new Map<LeftoverKind, LeftoverGroup>()
  for (const it of (preview?.items ?? []).filter((x) => x.kind !== 'store')) {
    const kind = leftoverKind(it)
    const g = groups.get(kind) ?? { kind, items: [], size: 0 }
    g.items.push(it)
    g.size += it.size
    groups.set(kind, g)
  }
  return [...groups.values()].toSorted((a, b) => b.size - a.size)
}

function leftoverLabel(i18n: I18n, kind: LeftoverKind): string {
  switch (kind) {
    case 'nexus':
      return i18n._(msg`Cached Nexus mod details`)
    case 'github':
      return i18n._(msg`Cached GitHub release lists`)
    case 'temp':
      return i18n._(msg`Unfinished downloads and temporary files`)
    default:
      return i18n._(msg`Other cached data`)
  }
}

function itemName(i18n: I18n, item: Item): string {
  if (item.name) {
    return item.name
  }
  const id = NEXUS_KEY.exec(item.key)?.[1]
  return id ? i18n._(msg`Nexus mod ${id}`) : item.key
}

function Row({
  checked,
  onToggle,
  title,
  detail,
  size,
}: {
  checked: boolean
  onToggle: (on: boolean) => void
  title: string
  detail: ReactNode
  size: number
}) {
  return (
    <Box
      component="label"
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 1.5,
        minHeight: 44,
        px: 1.5,
        borderRadius: '6px',
        cursor: 'pointer',
        '&:hover': { bgcolor: 'action.hover' },
      }}
    >
      <Checkbox size="small" checked={checked} onChange={(ev) => onToggle(ev.target.checked)} />
      <Box
        sx={{
          flex: 1,
          minWidth: 0,
          fontSize: 15,
          ...nowrap,
          overflow: 'hidden',
          textOverflow: 'ellipsis',
        }}
      >
        {title}
      </Box>
      <Box sx={{ flexShrink: 0, fontSize: 13, color: 'text.secondary', ...nowrap }}>{detail}</Box>
      <Box
        sx={{
          flexShrink: 0,
          width: 96,
          textAlign: 'right',
          fontSize: 15,
          fontWeight: 600,
          fontVariantNumeric: 'tabular-nums',
        }}
      >
        {formatBytes(size)}
      </Box>
    </Box>
  )
}

function Section({
  title,
  total,
  children,
}: {
  title: string
  total: number
  children: ReactNode
}) {
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column' }}>
      <Box
        sx={{
          display: 'flex',
          justifyContent: 'space-between',
          fontSize: 12,
          fontWeight: 700,
          letterSpacing: '0.06em',
          textTransform: 'uppercase',
          color: 'text.secondary',
          px: 1.5,
          pt: 1.5,
          pb: 0.5,
        }}
      >
        <span>{title}</span>
        <Box component="span" sx={{ fontVariantNumeric: 'tabular-nums' }}>
          {formatBytes(total)}
        </Box>
      </Box>
      {children}
    </Box>
  )
}

function useCleanupData(open: boolean) {
  const [store, setStore] = useState<{ unused: Sel[]; older: Sel[] } | null>(null)
  const [preview, setPreview] = useState<Preview | null>(null)
  const [error, setError] = useState(false)
  const load = useCallback(() => {
    setError(false)
    Promise.all([Report(), CleanupPreview()])
      .then(([rep, next]) => {
        setStore(removable(rep))
        setPreview(next)
      })
      .catch(() => setError(true))
  }, [])
  useEffect(() => {
    if (open) {
      load()
    }
  }, [open, load])
  return { store, preview, load, error }
}

function CleanupBody({
  store,
  leftovers,
  picked,
  setPicked,
  toggle,
}: {
  store: { unused: Sel[]; older: Sel[] } | null
  leftovers: LeftoverGroup[]
  picked: Set<string>
  setPicked: (next: Set<string>) => void
  toggle: (id: string, on: boolean) => void
}) {
  const { t, i18n } = useLingui()
  if (store === null) {
    return <Skeleton height={44} />
  }
  const items = [...store.unused, ...store.older]
  const allIds = [...items.map((s) => s.id), ...leftovers.map((g) => g.kind)]
  const itemRow = (s: Sel, detail: ReactNode) => (
    <Row
      key={s.id}
      checked={picked.has(s.id)}
      onToggle={(on) => toggle(s.id, on)}
      title={
        s.item.version ? `${itemName(i18n, s.item)} ${s.item.version}` : itemName(i18n, s.item)
      }
      detail={detail}
      size={s.item.size}
    />
  )
  const sum = (rows: Sel[]) => rows.reduce((n, s) => n + s.item.size, 0)
  if (allIds.length === 0) {
    return <Box sx={{ fontSize: 15, px: 1.5, py: 1.5 }}>{t`Nothing to clean up.`}</Box>
  }
  return (
    <>
      <Row
        checked={allIds.every((id) => picked.has(id))}
        onToggle={(on) => setPicked(on ? new Set(allIds) : new Set())}
        title={t`Select all`}
        detail=""
        size={sum(items) + leftovers.reduce((n, g) => n + g.size, 0)}
      />
      {store.unused.length > 0 ? (
        <Section title={t`Mods no profile uses`} total={sum(store.unused)}>
          {store.unused.map((s) =>
            itemRow(s, s.item.lastUsed ? <When value={s.item.lastUsed} /> : ''),
          )}
        </Section>
      ) : null}
      {store.older.length > 0 ? (
        <Section title={t`Older copies of the same mod`} total={sum(store.older)}>
          {store.older.map((s) => itemRow(s, t`a newer copy is kept`))}
        </Section>
      ) : null}
      {leftovers.length > 0 ? (
        <Section title={t`Leftover files`} total={leftovers.reduce((n, g) => n + g.size, 0)}>
          {leftovers.map((g) => (
            <Row
              key={g.kind}
              checked={picked.has(g.kind)}
              onToggle={(on) => toggle(g.kind, on)}
              title={leftoverLabel(i18n, g.kind)}
              detail={plural(g.items.length, { one: '# file', other: '# files' })}
              size={g.size}
            />
          ))}
        </Section>
      ) : null}
    </>
  )
}

function CleanupDialog({
  open,
  onClose,
  onChanged,
}: {
  open: boolean
  onClose: () => void
  onChanged: () => void
}) {
  const { t } = useLingui()
  const { store, preview, load, error } = useCleanupData(open)
  const [picked, setPicked] = useState<Set<string>>(new Set())
  const [confirm, setConfirm] = useState(false)
  const [busy, runCleanup] = usePending()
  const items = [...(store?.unused ?? []), ...(store?.older ?? [])]
  const leftovers = groupLeftovers(preview)
  const close = () => {
    setPicked(new Set())
    onClose()
  }
  const toggle = (id: string, on: boolean) =>
    setPicked((cur) => {
      const next = new Set(cur)
      if (on) {
        next.add(id)
      } else {
        next.delete(id)
      }
      return next
    })
  const chosenItems = items.filter((s) => picked.has(s.id))
  const chosenLeftovers = leftovers.filter((g) => picked.has(g.kind))
  const bytes =
    chosenItems.reduce((n, s) => n + s.item.size, 0) +
    chosenLeftovers.reduce((n, g) => n + g.size, 0)
  const run = () => {
    const byGame = new Map<string, string[]>()
    for (const s of chosenItems) {
      byGame.set(s.game, [...(byGame.get(s.game) ?? []), s.item.key])
    }
    const files = chosenLeftovers.flatMap((g) => g.items)
    runCleanup(() =>
      Promise.all([
        ...[...byGame.entries()].map(([game, keys]) => RemoveItems(game, keys)),
        ...(preview && files.length > 0
          ? [Cleanup({ ...preview, items: files, total: files.reduce((n, f) => n + f.size, 0) })]
          : []),
      ]).then(() => {
        setConfirm(false)
        setPicked(new Set())
        load()
        close()
        onChanged()
        useToasts.getState().push({ kind: 'success', title: t`Freed ${formatBytes(bytes)}` })
      }),
    )
  }
  return (
    <Dialog open={open} onClose={close} maxWidth="md" fullWidth={true} scroll="paper">
      <DialogTitle>{t`Clean up storage`}</DialogTitle>
      <DialogContent dividers={true} sx={{ display: 'flex', flexDirection: 'column', py: 0.5 }}>
        {error ? (
          <Box
            role="alert"
            sx={{ display: 'flex', alignItems: 'center', gap: 1, px: 1.5, py: 1.5, fontSize: 15 }}
          >
            {t`Could not load what can be cleaned up`}
            <Button variant="outlined" onClick={load}>
              {t`Retry`}
            </Button>
          </Box>
        ) : (
          <CleanupBody
            store={store}
            leftovers={leftovers}
            picked={picked}
            setPicked={setPicked}
            toggle={toggle}
          />
        )}
      </DialogContent>
      <DialogActions>
        <Box sx={{ flex: 1, pl: 1, fontSize: 15 }}>
          {picked.size > 0 ? t`${picked.size} selected · ${formatBytes(bytes)}` : ''}
        </Box>
        <Button onClick={close}>{t`Close`}</Button>
        <Button
          variant="contained"
          color="error"
          disabled={picked.size === 0}
          onClick={() => setConfirm(true)}
        >
          {t`Remove`}
        </Button>
      </DialogActions>
      <ConfirmDialog
        open={confirm}
        title={t`Remove the selected items?`}
        body={t`This frees ${formatBytes(bytes)}. Removed mods download again if a profile needs them later.`}
        confirmLabel={t`Remove`}
        color="error"
        busy={busy}
        onCancel={() => setConfirm(false)}
        onConfirm={run}
      />
    </Dialog>
  )
}

export { CleanupDialog }
