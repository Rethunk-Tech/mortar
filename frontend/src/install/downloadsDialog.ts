import { create } from 'zustand'
import type { DownloadArchive } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/archivesvc/models.ts'

export const useDownloadsDialog = create<{ open: boolean; setOpen: (open: boolean) => void }>(
  (set) => ({ open: false, setOpen: (open) => set({ open }) }),
)

export const openDownloadsDialog = () => useDownloadsDialog.getState().setOpen(true)

/** Newest first, narrowed to names containing the query. */
export function listArchives(
  archives: readonly DownloadArchive[],
  query: string,
): DownloadArchive[] {
  const needle = query.trim().toLowerCase()
  return archives
    .filter((a) => a.name.toLowerCase().includes(needle))
    .sort((a, b) => b.mtime - a.mtime)
}
