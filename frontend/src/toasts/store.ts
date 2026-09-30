import { create } from 'zustand'

const QUICK_MS = 5000
const SLOW_MS = 10_000
const MAX_SHOWN = 3

const timers = new Map<number, ReturnType<typeof setTimeout>>()

let nextId = 1

export type ToastKind = 'info' | 'success' | 'warning' | 'error'

export interface ToastInput {
  kind: ToastKind
  title: string
  body?: string
  detail?: string
  picture?: string
  action?: {
    label: string
    run: () => unknown
    profileId?: string
  }
}

export interface Toast extends ToastInput {
  id: number
}

// Failures stay longer: they are read, not glanced at.
export const lifetime = (kind: ToastKind) =>
  kind === 'info' || kind === 'success' ? QUICK_MS : SLOW_MS

export const useToasts = create<{
  toasts: Toast[]
  push: (toast: ToastInput) => number
  dismiss: (id: number) => void
  hold: (id: number) => void
  release: (id: number) => void
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
    push: (input) => {
      const id = nextId
      nextId += 1
      const kept = [...get().toasts, { ...input, id }]
      for (const gone of kept.slice(0, -MAX_SHOWN)) {
        clearTimeout(timers.get(gone.id))
        timers.delete(gone.id)
      }
      set({ toasts: kept.slice(-MAX_SHOWN) })
      arm(id, input.kind)
      return id
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
  }
})
