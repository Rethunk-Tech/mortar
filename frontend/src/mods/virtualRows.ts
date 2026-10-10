import { useVirtualizer } from '@tanstack/react-virtual'
import { type RefObject, useEffect, useLayoutEffect, useRef } from 'react'
import { isTypingTarget, type TypingTarget } from '../settings/shortcuts.ts'
import { TYPE_RESET_MS, typedKey, typedMatch } from '../shell/typeahead.ts'
import { toggleCollapsed } from './group.ts'
import type { ListRow } from './listColumns.ts'
import { modId } from './lookup.ts'

const GRID_MIN_CARD_PX = 300

const GRID_GAP_PX = 6

const GRID_PAD_X_PX = 16

const GRID_CARD_PX = 64

const GRID_CARD_COMPACT_PX = 50

function estimateVirtualSize<T>(row: VirtualRow<T>, lanePx: number): number {
  if (row.kind === 'header') {
    return GROUP_HEADER_PX
  }
  if (row.kind === 'lane') {
    return lanePx
  }
  return LIST_ROW_PX
}

interface Scroller {
  scrollToIndex: (index: number, opts?: { align: 'auto' }) => void
}

export const GROUP_HEADER_PX = 36

export const LIST_ROW_PX = 36

/** An optional file laid over a mod's main file, listed under the main file's row. */
export interface OverlayRow {
  key: string
  baseKey: string
  label: string
  enabled: boolean
  baseEnabled: boolean
}

export type VirtualRow<T> =
  | { kind: 'header'; key: string; groupKey: string; count: number }
  | { kind: 'row'; key: string; groupKey: string; item: T; stripe: boolean }
  | { kind: 'lane'; key: string; groupKey: string; items: readonly T[] }
  | { kind: 'overlay'; key: string; groupKey: string; overlay: OverlayRow }

export function gridColumnCount(width: number): number {
  if (width <= 0) {
    return 1
  }
  const inner = width - GRID_PAD_X_PX * 2
  return Math.max(1, Math.floor((inner + GRID_GAP_PX) / (GRID_MIN_CARD_PX + GRID_GAP_PX)))
}

export function gridLanePx(compact: boolean): number {
  return (compact ? GRID_CARD_COMPACT_PX : GRID_CARD_PX) + GRID_GAP_PX
}

export function flattenModGroups<T>(
  groups: readonly { key: string; items: readonly T[] }[],
  opts: {
    grouped: boolean
    collapsed: Readonly<Record<string, boolean>>
    idOf: (item: T) => string
    columns?: number
  },
): VirtualRow<T>[] {
  const { grouped, collapsed, idOf, columns } = opts
  const out: VirtualRow<T>[] = []
  for (const group of groups) {
    if (grouped) {
      out.push({
        kind: 'header',
        key: `h:${group.key}`,
        groupKey: group.key,
        count: group.items.length,
      })
    }
    // Without grouping there is no header to reopen a group, so a collapse saved under another grouping (the
    // ungrouped list shares the empty key with Uncategorised) must not hide the list.
    if (!grouped || collapsed[group.key] !== true) {
      if (columns === undefined) {
        for (const [i, item] of group.items.entries()) {
          out.push({
            kind: 'row',
            key: `r:${idOf(item)}`,
            groupKey: group.key,
            item,
            stripe: i % 2 === 1,
          })
        }
      } else {
        const cols = Math.max(1, columns)
        for (let i = 0; i < group.items.length; i += cols) {
          out.push({
            kind: 'lane',
            key: `l:${group.key}:${i}`,
            groupKey: group.key,
            items: group.items.slice(i, i + cols),
          })
        }
      }
    }
  }
  return out
}

export function groupKeyHolding<T>(
  groups: readonly { key: string; items: readonly T[] }[],
  match: (item: T) => boolean,
): string | undefined {
  for (const group of groups) {
    if (group.items.some(match)) {
      return group.key
    }
  }
  return undefined
}

export function virtualIndexOf<T>(
  rows: readonly VirtualRow<T>[],
  id: string,
  idOf: (item: T) => string,
): number {
  return rows.findIndex((row) => {
    if (row.kind === 'row') {
      return idOf(row.item) === id
    }
    if (row.kind === 'lane') {
      return row.items.some((item) => idOf(item) === id)
    }
    return false
  })
}

export function orderedModIds<T>(
  rows: readonly VirtualRow<T>[],
  idOf: (item: T) => string,
): string[] {
  const ids: string[] = []
  for (const row of rows) {
    if (row.kind === 'row') {
      ids.push(idOf(row.item))
    } else if (row.kind === 'lane') {
      for (const item of row.items) {
        ids.push(idOf(item))
      }
    }
  }
  return ids
}

// Moves `delta` places without wrapping, stopping at the first or last id; grids step by a row's width.
export function stepId(ids: readonly string[], current: string, delta: number): string | undefined {
  const i = ids.indexOf(current)
  if (i < 0) {
    return undefined
  }
  return ids[Math.min(ids.length - 1, Math.max(0, i + delta))]
}

export function useModVirtual<T>(items: readonly VirtualRow<T>[], lanePx: number) {
  const parentRef = useRef<HTMLDivElement>(null)
  const virtualizer = useVirtualizer({
    count: items.length,
    getScrollElement: () => parentRef.current,
    estimateSize: (index) => {
      const row = items[index]
      return row ? estimateVirtualSize(row, lanePx) : LIST_ROW_PX
    },
    overscan: 10,
    getItemKey: (index) => items[index]?.key ?? index,
  })
  return { parentRef, virtualizer }
}

export function typeaheadChar(e: {
  key: string
  ctrlKey: boolean
  altKey: boolean
  metaKey: boolean
  target: EventTarget | null
}): string | undefined {
  if (e.ctrlKey || e.altKey || e.metaKey || e.key.length !== 1) {
    return undefined
  }
  if (!typedKey(e.key)) {
    return undefined
  }
  if (isTypingTarget(e.target as TypingTarget | null)) {
    return undefined
  }
  return e.key
}

export function typeaheadQuery(prev: string, at: number, key: string, now: number): string {
  if (now - at > TYPE_RESET_MS) {
    return key
  }
  return prev + key
}

export const listRowId = (row: ListRow) => modId(row.mod)

export interface ModView<T> {
  items: readonly VirtualRow<T>[]
  idOf: (item: T) => string
  virtualizer: Scroller
  parentRef: RefObject<HTMLElement | null>
}

// Scrolls the row into the virtual window, then focuses it once it has rendered.
export function focusModAt<T>(view: ModView<T>, id: string) {
  const { items, idOf, virtualizer, parentRef } = view
  const idx = virtualIndexOf(items, id, idOf)
  if (idx >= 0) {
    virtualizer.scrollToIndex(idx, { align: 'auto' })
  }
  requestAnimationFrame(() => {
    requestAnimationFrame(() => {
      parentRef.current?.querySelector<HTMLElement>(`[data-mod-id="${CSS.escape(id)}"]`)?.focus()
    })
  })
}

// Brings the open detail's row into view, expanding its group first when collapsed.
export function useModReveal<T>(opts: {
  detailId: string
  groups: readonly { key: string; items: readonly T[] }[]
  items: readonly VirtualRow<T>[]
  idOf: (item: T) => string
  collapsed: Record<string, boolean>
  setCollapsed: (fn: (cur: Record<string, boolean>) => Record<string, boolean>) => void
  gameId: string
  virtualizer: Scroller
}) {
  const { detailId, groups, items, idOf, collapsed, setCollapsed, gameId, virtualizer } = opts
  const lastReveal = useRef('')
  useLayoutEffect(() => {
    if (!detailId) {
      return
    }
    const held = groupKeyHolding(groups, (row) => idOf(row) === detailId)
    if (held !== undefined && collapsed[held] === true) {
      setCollapsed((cur) => toggleCollapsed(gameId, cur, held, false))
      return
    }
    const idx = virtualIndexOf(items, detailId, idOf)
    const token = `${detailId}:${idx}`
    if (lastReveal.current === token || idx < 0) {
      return
    }
    lastReveal.current = token
    virtualizer.scrollToIndex(idx, { align: 'auto' })
  }, [collapsed, detailId, gameId, groups, idOf, items, setCollapsed, virtualizer])
}

function modsIn<T>(items: readonly VirtualRow<T>[]): T[] {
  const mods: T[] = []
  for (const row of items) {
    if (row.kind === 'row') {
      mods.push(row.item)
    } else if (row.kind === 'lane') {
      mods.push(...row.items)
    }
  }
  return mods
}

export function useModTypeahead<T>(opts: {
  items: readonly VirtualRow<T>[]
  nameOf: (item: T) => string
  idOf: (item: T) => string
  virtualizer: Scroller
  parentRef: RefObject<HTMLElement | null>
}) {
  const { items, nameOf, idOf, virtualizer, parentRef } = opts
  const buf = useRef({ text: '', at: 0 })
  useEffect(() => {
    const root = parentRef.current
    if (!root) {
      return
    }
    const onKey = (e: KeyboardEvent) => {
      const now = Date.now()
      const ch = typeaheadChar(e)
      if (ch === undefined) {
        // A space inside a name being typed would otherwise act on the row that typing just focused.
        const typing = isTypingTarget(e.target as TypingTarget | null)
        if (e.key === ' ' && !typing && buf.current.text && now - buf.current.at <= TYPE_RESET_MS) {
          e.preventDefault()
        }
        return
      }
      const active = document.activeElement
      if (active !== root && !root.contains(active)) {
        return
      }
      e.preventDefault()
      const text = typeaheadQuery(buf.current.text, buf.current.at, ch, now)
      buf.current = { text, at: now }
      const hit = typedMatch(modsIn(items), text, nameOf)
      if (!hit) {
        return
      }
      const id = idOf(hit)
      focusModAt({ items, idOf, virtualizer, parentRef }, id)
    }
    root.addEventListener('keydown', onKey)
    return () => root.removeEventListener('keydown', onKey)
  }, [idOf, items, nameOf, parentRef, virtualizer])
}
