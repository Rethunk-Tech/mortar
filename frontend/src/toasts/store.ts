import { create } from 'zustand'
import { HISTORY_CAP, type HistoryActionState, prependHistory } from './history.ts'

const QUICK_MS = 5000
const SLOW_MS = 10_000
const MAX_SHOWN = 3

const timers = new Map<number, ReturnType<typeof setTimeout>>()

let nextId = 1

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
}

export interface Toast extends ToastInput {
  id: number
}

export interface ToastHistoryItem {
  id: number
  at: number
  kind: ToastKind
  title: string
  body?: string
  detail?: string
  picture?: string
  count?: number
  action?: ToastAction
}

// Failures stay longer: they are read, not glanced at.
export const lifetime = (kind: ToastKind) =>
  kind === 'info' || kind === 'success' ? QUICK_MS : SLOW_MS

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
  const arm = (id: number, kind: ToastKind) => {
    clearTimeout(timers.get(id))
    timers.set(
      id,
      setTimeout(() => get().dismiss(id), lifetime(kind)),
    )
  }
  return {
    toasts: [],
    history: [],
    unread: 0,
    historyOpen: false,
    push: (input) => {
      const now = Date.now()
      const [previous] = get().history
      if (previous && previous.title === input.title && now - previous.at < QUICK_MS) {
        const count = (previous.count ?? 1) + 1
        const merged = { ...previous, at: now, count }
        set((s) => ({
          history: [merged, ...s.history.slice(1)],
          toasts: s.toasts.map((toast) =>
            toast.title === input.title ? { ...toast, count } : toast,
          ),
        }))
        return previous.id
      }
      const id = nextId
      nextId += 1
      const kept = [...get().toasts, { ...input, id }]
      for (const gone of kept.slice(0, -MAX_SHOWN)) {
        clearTimeout(timers.get(gone.id))
        timers.delete(gone.id)
      }
      const item: ToastHistoryItem = {
        id,
        at: now,
        kind: input.kind,
        title: input.title,
        ...(input.body === undefined ? {} : { body: input.body }),
        ...(input.detail === undefined ? {} : { detail: input.detail }),
        ...(input.picture === undefined ? {} : { picture: input.picture }),
        ...(input.action === undefined ? {} : { action: input.action }),
      }
      set({
        toasts: kept.slice(-MAX_SHOWN),
        history: prependHistory(get().history, item),
        unread: Math.min(HISTORY_CAP, get().unread + 1),
      })
      arm(id, input.kind)
      return id
    },
    update: (id, input) => {
      set((s) => ({
        toasts: s.toasts.map((toast) => (toast.id === id ? { ...toast, ...input } : toast)),
        history: s.history.map((toast) => (toast.id === id ? { ...toast, ...input } : toast)),
      }))
    },
    dismiss: (id) => {
      clearTimeout(timers.get(id))
      timers.delete(id)
      set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) }))
    },
    // Hovering pauses the countdown; leaving restarts it in full.
    hold: (id) => clearTimeout(timers.get(id)),
    release: (id) => {
      const toast = get().toasts.find((t) => t.id === id)
      if (toast) {
        arm(id, toast.kind)
      }
    },
    markRead: () => set({ unread: 0 }),
    clearHistory: () => set({ history: [], unread: 0 }),
    setHistoryOpen: (open) => set({ historyOpen: open, ...(open ? { unread: 0 } : {}) }),
  }
})
