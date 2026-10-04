import { create } from 'zustand'

// The release the What's new dialog shows; fallback is the notes the updater already holds, used when GitHub is unreachable.
export const useWhatsNew = create<{ version: string; fallback: string } | null>(() => null)

export function showWhatsNew(version: string, fallback = ''): void {
  useWhatsNew.setState({ version, fallback }, true)
}
