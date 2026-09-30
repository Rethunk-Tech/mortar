import { create } from 'zustand'

const AUTO_DISMISS_MS = 5000

let nextId = 1

export type ToastKind = 'info' | 'success' | 'warning' | 'error'

export interface ToastInput {
  kind: ToastKind
  title: string
  body?: string
  action?: { label: string; run: () => void }
}

export interface Toast extends ToastInput {
  id: number
}

export const autoDismisses = (kind: ToastKind) => kind === 'info' || kind === 'success'

export const useToasts = create<{
  toasts: Toast[]
  push: (toast: ToastInput) => number
  dismiss: (id: number) => void
}>((set, get) => ({
  toasts: [],
  push: (input) => {
    const id = nextId
    nextId += 1
    set((s) => ({ toasts: [...s.toasts, { ...input, id }] }))
    if (autoDismisses(input.kind)) {
      setTimeout(() => get().dismiss(id), AUTO_DISMISS_MS)
    }
    return id
  },
  dismiss: (id) => set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) })),
}))
