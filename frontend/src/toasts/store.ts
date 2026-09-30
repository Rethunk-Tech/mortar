import { create } from 'zustand'

export type ToastKind = 'info' | 'success' | 'warning' | 'error'

export type ToastInput = {
  kind: ToastKind
  title: string
  body?: string
  action?: { label: string; run: () => void }
}

export type Toast = ToastInput & { id: number }

const AUTO_DISMISS_MS = 5000

export const autoDismisses = (kind: ToastKind) => kind === 'info' || kind === 'success'

let nextId = 1

export const useToasts = create<{
  toasts: Toast[]
  push: (toast: ToastInput) => number
  dismiss: (id: number) => void
}>((set, get) => ({
  toasts: [],
  push: (input) => {
    const id = nextId++
    set((s) => ({ toasts: [...s.toasts, { ...input, id }] }))
    if (autoDismisses(input.kind)) {
      setTimeout(() => get().dismiss(id), AUTO_DISMISS_MS)
    }
    return id
  },
  dismiss: (id) => set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) })),
}))
