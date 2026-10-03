import { useVirtualizer } from '@tanstack/react-virtual'
import { type RefObject, useEffect, useRef } from 'react'

const TYPEAHEAD_LETTER = /^\p{L}$/u
export const TYPEAHEAD_MS = 500

export const LIST_ROW_PX = 36
export const GROUP_HEADER_PX = 36
export const GRID_MIN_CARD_PX = 300
export const GRID_GAP_PX = 6
export const GRID_PAD_X_PX = 16
export const GRID_CARD_PX = 64
export const GRID_CARD_COMPACT_PX = 50

export type VirtualRow<T> =
  | { kind: 'header'; key: string; groupKey: string; count: number }
  | { kind: 'row'; key: string; groupKey: string; item: T; stripe: boolean }
  | { kind: 'lane'; key: string; groupKey: string; items: readonly T[] }

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
    if (collapsed[group.key] !== true) {
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

export function neighborId(
  ids: readonly string[],
  current: string,
  dir: -1 | 1,
): string | undefined {
  if (ids.length === 0) {
    return undefined
  }
  const i = ids.indexOf(current)
  if (i < 0) {
    return undefined
  }
  return ids[(i + dir + ids.length) % ids.length]
}

export function estimateVirtualSize<T>(row: VirtualRow<T>, lanePx: number): number {
  if (row.kind === 'header') {
    return GROUP_HEADER_PX
  }
  if (row.kind === 'lane') {
    return lanePx
  }
  return LIST_ROW_PX
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
  if (!TYPEAHEAD_LETTER.test(e.key)) {
    return undefined
  }
  const el = e.target as { tagName?: string; isContentEditable?: boolean } | null
  if (el?.tagName === 'INPUT' || el?.tagName === 'TEXTAREA' || el?.isContentEditable) {
    return undefined
  }
  return e.key
}

export function typeaheadQuery(prev: string, at: number, key: string, now: number): string {
  if (now - at > TYPEAHEAD_MS) {
    return key
  }
  return prev + key
}

export function firstNamePrefix<T>(
  items: readonly T[],
  query: string,
  nameOf: (item: T) => string,
): T | undefined {
  const q = query.toLocaleLowerCase()
  if (!q) {
    return undefined
  }
  return items.find((item) => nameOf(item).toLocaleLowerCase().startsWith(q))
}

export function useModTypeahead<T>(opts: {
  items: readonly VirtualRow<T>[]
  nameOf: (item: T) => string
  idOf: (item: T) => string
  virtualizer: { scrollToIndex: (index: number, opts?: { align: 'auto' }) => void }
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
      const ch = typeaheadChar(e)
      if (ch === undefined) {
        return
      }
      const active = document.activeElement
      if (active !== root && !root.contains(active)) {
        return
      }
      e.preventDefault()
      const now = Date.now()
      const text = typeaheadQuery(buf.current.text, buf.current.at, ch, now)
      buf.current = { text, at: now }
      const mods: T[] = []
      for (const row of items) {
        if (row.kind === 'row') {
          mods.push(row.item)
        } else if (row.kind === 'lane') {
          mods.push(...row.items)
        }
      }
      const hit = firstNamePrefix(mods, text, nameOf)
      if (!hit) {
        return
      }
      const id = idOf(hit)
      const idx = virtualIndexOf(items, id, idOf)
      if (idx >= 0) {
        virtualizer.scrollToIndex(idx, { align: 'auto' })
      }
      requestAnimationFrame(() => {
        parentRef.current?.querySelector<HTMLElement>(`[data-mod-id="${CSS.escape(id)}"]`)?.focus()
      })
    }
    root.addEventListener('keydown', onKey)
    return () => root.removeEventListener('keydown', onKey)
  }, [idOf, items, nameOf, parentRef, virtualizer])
}
