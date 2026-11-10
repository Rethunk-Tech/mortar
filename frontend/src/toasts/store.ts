import { readStored, writeStored } from '@rethunk/react-load-hooks'
import { create } from 'zustand'
import { HISTORY_CAP, type HistoryActionState, prependHistory } from './history.ts'

const QUICK_MS = 5000
const SLOW_MS = 10_000
const MAX_SHOWN = 3
// How long a sticky toast that finished well stays up.
const DONE_MS = 8000

const timers = new Map<number, ReturnType<typeof setTimeout>>()

let nextId = 1
// The newest notification a repeat still folds into. A timer closes that window, never a second Date.now() read: the
// clock's resolution makes two pushes in one millisecond the same, and a step either way would merge a repeat that
// arrived later into the older entry, or start a second entry for a repeat that is still within the window.
let repeatOf: number | undefined
let repeatTimer: ReturnType<typeof setTimeout> | undefined

const SAVED_KEY = 'mortar.toastHistory'
const SAVED_CAP = 100
const KINDS: readonly string[] = ['info', 'success', 'warning', 'error']

function isSavedItem(v: unknown): v is ToastHistoryItem {
  if (typeof v !== 'object' || v === null) {
    return false
  }
  const o = v as Record<string, unknown>
  return typeof o.at === 'number' && typeof o.title === 'string' && KINDS.includes(String(o.kind))
}

// The previous session's notifications come back without their actions (those closed over live state) and
// with negative ids so they never collide with this session's.
function loadSaved(): ToastHistoryItem[] {
  return readStored(SAVED_KEY, [], Array.isArray)
    .filter(isSavedItem)
    .map((item, i) => ({ ...item, id: -(i + 1) }))
}

function saveHistory(history: ToastHistoryItem[]) {
  writeStored(
    SAVED_KEY,
    history.slice(0, SAVED_CAP).map(({ action: _, ...rest }) => rest),
  )
}

// Failures stay longer: they are read, not glanced at.
const lifetime = (kind: ToastKind) => (kind === 'info' || kind === 'success' ? QUICK_MS : SLOW_MS)

// The profiles store answers with the history event its last change returned; toasts cannot import it back.
let latestChange: (profileId: string) => string = () => ''
// The settings store answers whether a success pops up; it imports this store, so it cannot be imported back.
let popSuccess: () => boolean = () => false

// A success with nothing to do about it only goes to the notification list unless the player asked for pop-ups. One
// with an action (Undo, Open) still pops, since the action is worth most right away, and so does a sticky progress
// notice, which the player is watching.
const pops = (input: ToastInput) =>
  input.kind !== 'success' || input.action !== undefined || input.sticky === true || popSuccess()

const changesOf = (
  input: Pick<ToastInput, 'action' | 'changes'>,
  earlier: readonly string[] = [],
) => {
  const latest = input.action?.profileId ? latestChange(input.action.profileId) : ''
  const all = new Set([...earlier, ...(input.changes ?? []), ...(latest ? [latest] : [])])
  all.delete('')
  return [...all]
}

export type ToastKind = 'info' | 'success' | 'warning' | 'error'

export interface ToastAction {
  label: string
  run: () => unknown
  profileId?: string
  live?: () => HistoryActionState
}

export interface ToastInput {
  kind: ToastKind
  title: string
  body?: string
  detail?: string
  picture?: string
  count?: number
  action?: ToastAction
  // Stays until dismissed, or until an update sets this false, which starts its countdown.
  sticky?: boolean
  // History events a change reports that the profiles store cannot answer for: those a batch wrote.
  changes?: string[]
  // A standing state rather than an event (how many updates wait): a newer notice on the same topic replaces the older
  // one in the history instead of stacking beside it.
  topic?: string
}

export interface Toast extends ToastInput {
  id: number
}

export interface ToastHistoryItem {
  id: number
  // Shown in the panel only. The order of the list is the store's own push order, never this clock reading.
  at: number
  // The history events this notification reports, so the panel lists each change once.
  changes?: string[]
  kind: ToastKind
  title: string
  body?: string
  detail?: string
  picture?: string
  count?: number
  action?: ToastAction
  topic?: string
}

function historyItem(
  id: number,
  at: number,
  input: ToastInput,
  changes: string[],
): ToastHistoryItem {
  return {
    id,
    at,
    ...(changes.length > 0 ? { changes } : {}),
    kind: input.kind,
    title: input.title,
    ...(input.body === undefined ? {} : { body: input.body }),
    ...(input.detail === undefined ? {} : { detail: input.detail }),
    ...(input.picture === undefined ? {} : { picture: input.picture }),
    ...(input.action === undefined ? {} : { action: input.action }),
    ...(input.topic === undefined ? {} : { topic: input.topic }),
  }
}

const withoutTopic = (history: ToastHistoryItem[], topic: string | undefined) =>
  topic === undefined ? history : history.filter((h) => h.topic !== topic)

export const useToasts = create<{
  toasts: Toast[]
  history: ToastHistoryItem[]
  unread: number
  push: (toast: ToastInput) => number
  update: (id: number, toast: Partial<ToastInput>) => void
  dismiss: (id: number) => void
  hold: (id: number) => void
  release: (id: number) => void
  markRead: () => void
  clearHistory: () => void
  historyOpen: boolean
  setHistoryOpen: (open: boolean) => void
}>((set, get) => {
  const arm = (id: number, kind: ToastKind, ms = lifetime(kind)) => {
    clearTimeout(timers.get(id))
    timers.set(
      id,
      setTimeout(() => get().dismiss(id), ms),
    )
  }
  const openRepeat = (id: number) => {
    clearTimeout(repeatTimer)
    repeatOf = id
    repeatTimer = setTimeout(() => {
      repeatOf = undefined
      repeatTimer = undefined
    }, QUICK_MS)
  }
  return {
    toasts: [],
    history: loadSaved(),
    unread: 0,
    historyOpen: false,
    push: (input) => {
      const now = Date.now()
      const [previous] = get().history
      if (previous && previous.id === repeatOf && previous.title === input.title) {
        const count = (previous.count ?? 1) + 1
        const changes = changesOf(input, previous.changes)
        const merged = { ...previous, at: now, count, ...(changes.length > 0 ? { changes } : {}) }
        const shown = get().toasts.some((toast) => toast.title === input.title)
        const popped = shown || pops(input)
        set((s) => {
          let toasts = s.toasts
          if (shown) {
            toasts = s.toasts.map((toast) =>
              toast.title === input.title ? { ...toast, count } : toast,
            )
          } else if (popped) {
            toasts = [...s.toasts, { ...input, id: previous.id, count }].slice(-MAX_SHOWN)
          }
          return { history: [merged, ...s.history.slice(1)], toasts }
        })
        saveHistory(get().history)
        if (popped && !(shown || input.sticky)) {
          arm(previous.id, input.kind)
        }
        openRepeat(previous.id)
        return previous.id
      }
      const id = nextId
      nextId += 1
      const popped = pops(input)
      const kept = popped ? [...get().toasts, { ...input, id }] : get().toasts
      for (const gone of kept.slice(0, -MAX_SHOWN)) {
        clearTimeout(timers.get(gone.id))
        timers.delete(gone.id)
      }
      set({
        toasts: kept.slice(-MAX_SHOWN),
        history: prependHistory(
          withoutTopic(get().history, input.topic),
          historyItem(id, now, input, changesOf(input)),
        ),
        unread: Math.min(HISTORY_CAP, get().unread + 1),
      })
      saveHistory(get().history)
      if (popped && !input.sticky) {
        arm(id, input.kind)
      }
      openRepeat(id)
      return id
    },
    update: (id, input) => {
      set((s) => ({
        toasts: s.toasts.some((toast) => toast.id === id)
          ? s.toasts.map((toast) => (toast.id === id ? { ...toast, ...input } : toast))
          : s.toasts,
        history: s.history.map((item) => {
          if (item.id !== id) {
            return item
          }
          const changes = changesOf(input, item.changes)
          return { ...item, ...input, ...(changes.length > 0 ? { changes } : {}) }
        }),
      }))
      saveHistory(get().history)
      if (input.sticky === false) {
        arm(id, input.kind ?? 'info', DONE_MS)
      }
    },
    dismiss: (id) => {
      clearTimeout(timers.get(id))
      timers.delete(id)
      // A toast already gone leaves the list as it is, so the host does not redraw for nothing.
      set((s) =>
        s.toasts.some((t) => t.id === id) ? { toasts: s.toasts.filter((t) => t.id !== id) } : s,
      )
    },
    // Hovering pauses the countdown; leaving restarts it in full.
    hold: (id) => clearTimeout(timers.get(id)),
    release: (id) => {
      const toast = get().toasts.find((t) => t.id === id)
      if (toast && !toast.sticky) {
        arm(id, toast.kind)
      }
    },
    markRead: () => set({ unread: 0 }),
    clearHistory: () => {
      set({ history: [], unread: 0 })
      saveHistory([])
    },
    setHistoryOpen: (open) => set({ historyOpen: open, ...(open ? { unread: 0 } : {}) }),
  }
})

export function setLatestChange(f: (profileId: string) => string) {
  latestChange = f
}

export function setPopSuccess(f: () => boolean) {
  popSuccess = f
}
