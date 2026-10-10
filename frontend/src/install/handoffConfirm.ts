import { create } from 'zustand'

/** A saved file waiting for the player's answer to "Install <file> as from <page>?". */
export interface HandoffAsk {
  file: string
  page: string
  answer: (asFromPage: boolean) => void
}

// Asks are answered in order, one dialog at a time, so several files saved from one page each get their question.
export const useHandoffAsk = create<{
  queue: HandoffAsk[]
  push: (ask: HandoffAsk) => void
  shift: () => void
}>((set) => ({
  queue: [],
  push: (ask) => set((s) => ({ queue: [...s.queue, ask] })),
  shift: () => set((s) => ({ queue: s.queue.slice(1) })),
}))

const baseName = (path: string) => path.split(/[\\/]/).pop() ?? path

/** Whether to record the saved file against the page it was saved from; false installs it as a plain local file. */
export const askHandoff = (file: string, page: string) =>
  new Promise<boolean>((resolve) =>
    useHandoffAsk.getState().push({ file: baseName(file), page, answer: resolve }),
  )
